// Package serverargs turns a dedicated server profile's settings (world name, port, password) into the server's
// launch arguments. The catalog says which arguments a server takes; the profile holds their values in server.json.
package serverargs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// File is the profile's server settings, a JSON object of setting id to value.
const File = "server.json"

// ConfigName is the config file Mortar renders in the profile for a server configured by file.
const ConfigName = "server.cfg"

// Load reads the profile's server settings; none are set when the file is missing or unreadable.
func Load(profileDir string) map[string]string {
	values := map[string]string{}
	if b, err := safefs.ReadFile(filepath.Join(profileDir, File)); err == nil {
		_ = json.Unmarshal(b, &values)
	}
	return values
}

// Set stores one setting after checking it against the catalog; an empty value goes back to the default.
func Set(profileDir string, spec components.ServerSpec, id, value string) error {
	var arg *components.ServerArg
	for i := range spec.Args {
		if spec.Args[i].ID == id {
			arg = &spec.Args[i]
		}
	}
	if arg == nil {
		return fmt.Errorf("this server has no setting %q", id)
	}
	if value != "" && !arg.Matches(value) {
		return fmt.Errorf("%s does not accept that value", arg.Name)
	}
	values := Load(profileDir)
	if value == "" {
		delete(values, id)
	} else {
		values[id] = value
	}
	// The file can hold a server password; WriteJSON creates it 0600.
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(filepath.Join(profileDir, File), values)
}

// value is the setting's value (the profile's, else the default) and whether the argument applies. An error names the
// setting and never the value.
func value(a components.ServerArg, values map[string]string) (string, bool, error) {
	v := values[a.ID]
	if v == "" {
		v = a.Default
	}
	switch {
	case v == "" && a.Required:
		return "", false, fmt.Errorf("set %s for this server first", a.Name)
	case v == "":
		return "", false, nil
	case !a.Matches(v):
		return "", false, errors.New(a.Name + " is not valid for this server")
	}
	return v, true, nil
}

// Args is the server's command line after the executable: the fixed arguments, then each flag setting's flag and
// value. A setting the profile leaves unset takes its default; one with neither is left out, or fails when required.
func Args(spec components.ServerSpec, values map[string]string) ([]string, error) {
	out := append([]string{}, spec.Fixed...)
	for _, a := range spec.Args {
		v, ok, err := value(a, values)
		if err != nil {
			return nil, err
		}
		if ok && a.Flag != "" {
			out = append(out, a.Flag, v)
		}
	}
	return out, nil
}

// Config is the server's config file: one `cvar "value";` line for each cvar setting, numbers unquoted.
func Config(spec components.ServerSpec, values map[string]string) (string, error) {
	var b strings.Builder
	for _, a := range spec.Args {
		v, ok, err := value(a, values)
		if err != nil {
			return "", err
		}
		if !ok || a.Cvar == "" {
			continue
		}
		if !numeric.MatchString(v) {
			v = `"` + v + `"`
		}
		fmt.Fprintf(&b, "%s %s;\n", a.Cvar, v)
	}
	return b.String(), nil
}

var numeric = regexp.MustCompile(`^[0-9]+$`)
