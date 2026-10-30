package pack

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	tstore "github.com/Rethunk-Tech/go-thunderstore"

	"github.com/Rethunk-Tech/mortar/internal/deps"
)

// Modpack reads a Thunderstore modpack package: a package zip whose manifest dependencies are the mod list and
// whose config/ folder holds the pack's settings. A package that ships code (a .dll) is a mod, not a modpack.
type Modpack struct{}

// ID names the format.
func (Modpack) ID() string { return "thunderstore-modpack" }

func readModpack(path string) (tstore.Manifest, map[string][]byte, bool) {
	data, err := readCapped(path, maxInput)
	if err != nil {
		return tstore.Manifest{}, nil, false
	}
	files, err := readZip(data)
	if err != nil {
		return tstore.Manifest{}, nil, false
	}
	m, err := tstore.ParseManifest(files["manifest.json"])
	if err != nil || len(m.Dependencies) == 0 {
		return tstore.Manifest{}, nil, false
	}
	for name := range files {
		if strings.EqualFold(filepath.Ext(name), ".dll") {
			return tstore.Manifest{}, nil, false
		}
	}
	return m, files, true
}

// Detect accepts a .zip path whose manifest has dependencies and no code.
func (Modpack) Detect(in Input) bool {
	if !strings.EqualFold(filepath.Ext(in.Path), ".zip") {
		return false
	}
	_, _, ok := readModpack(in.Path)
	return ok
}

// Parse lists the dependencies as packages and the zip's config/ as Configs.
func (Modpack) Parse(_ context.Context, in Input) (Draft, error) {
	m, files, ok := readModpack(in.Path)
	if !ok {
		return Draft{}, errors.New("not a Thunderstore modpack")
	}
	d := Draft{Name: m.Name, Configs: filesUnder(files, "config/")}
	for _, dep := range m.Dependencies {
		pkg, version, ok := deps.ThunderstoreRef(dep)
		if !ok || !nativeID.MatchString(pkg) || !versionNumber.MatchString(version) {
			return Draft{}, fmt.Errorf("malformed dependency %q", dep)
		}
		d.Packages = append(d.Packages, Ref{Source: thunderstore, Native: pkg, Version: version})
	}
	return d, nil
}
