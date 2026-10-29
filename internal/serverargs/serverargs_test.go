package serverargs

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

var spec = components.ServerSpec{
	Clients: "valheim",
	Fixed:   []string{"-nographics"},
	Args: []components.ServerArg{
		{ID: "world", Name: "World name", Flag: "-world", Required: true},
		{ID: "port", Name: "Port", Flag: "-port", Default: "2456", Pattern: `[0-9]{1,5}`},
		{ID: "password", Name: "Password", Flag: "-password", Secret: true, Pattern: `.{5,}`},
	},
}

func TestArgsFillDefaultsAndLeaveOutUnsetOptions(t *testing.T) {
	got, err := Args(spec, map[string]string{"world": "My World"})
	want := []string{"-nographics", "-world", "My World", "-port", "2456"}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("Args = %v, %v; want %v", got, err, want)
	}
	got, _ = Args(spec, map[string]string{"world": "W", "port": "3000", "password": "hunter22"})
	if !slices.Equal(got, []string{"-nographics", "-world", "W", "-port", "3000", "-password", "hunter22"}) {
		t.Fatalf("Args = %v", got)
	}
}

func TestArgsRefuseAMissingRequiredOrBadValueWithoutEchoingIt(t *testing.T) {
	if _, err := Args(spec, nil); err == nil || err.Error() != "set World name for this server first" {
		t.Fatalf("missing world: %v", err)
	}
	_, err := Args(spec, map[string]string{"world": "W", "password": "abc"})
	if err == nil || err.Error() != "Password is not valid for this server" {
		t.Fatalf("short password: %v", err)
	}
}

func TestSetValidatesStoresAndClears(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "profile")
	if err := Set(dir, spec, "nope", "x"); err == nil {
		t.Fatal("unknown setting accepted")
	}
	if err := Set(dir, spec, "port", "abc"); err == nil {
		t.Fatal("bad port accepted")
	}
	if err := Set(dir, spec, "world", "Dedicated"); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir)["world"]; got != "Dedicated" {
		t.Fatalf("world = %q", got)
	}
	// Windows reports every file as 0666; only POSIX modes can show the file is private.
	if info, err := os.Stat(filepath.Join(dir, File)); err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode: %v %v", info, err)
	}
	if err := Set(dir, spec, "world", ""); err != nil || len(Load(dir)) != 0 {
		t.Fatalf("clear: %v %v", err, Load(dir))
	}
}

func TestConfigRendersCvarsQuotedUnlessNumeric(t *testing.T) {
	cfgSpec := components.ServerSpec{Clients: "x", ConfigFile: "Data/Config/server.cfg", Args: []components.ServerArg{
		{ID: "name", Name: "Name", Cvar: "sv_hostname", Default: "Mortar server"},
		{ID: "port", Name: "Port", Cvar: "sv_port", Default: "27015"},
		{ID: "password", Name: "Password", Cvar: "sv_password"},
		{ID: "flag", Name: "Flag", Flag: "-x", Default: "1"},
	}}
	got, err := Config(cfgSpec, map[string]string{"password": "pw"})
	want := "sv_hostname \"Mortar server\";\nsv_port 27015;\nsv_password \"pw\";\n"
	if err != nil || got != want {
		t.Fatalf("Config = %q, %v; want %q", got, err, want)
	}
	if args, _ := Args(cfgSpec, nil); !slices.Equal(args, []string{"-x", "1"}) {
		t.Fatalf("cvars leaked onto the command line: %v", args)
	}
}
