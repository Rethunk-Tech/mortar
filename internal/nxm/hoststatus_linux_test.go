//go:build linux

package nxm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/nativehost"
)

func TestNativeHostStatusOkMissingAndStale(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".config")
	mortar := filepath.Join(home, "mortar")
	if err := os.WriteFile(mortar, []byte("bin"), 0o600); err != nil {
		t.Fatal(err)
	}
	chromeDir := filepath.Join(cfg, "google-chrome")
	if err := os.MkdirAll(filepath.Join(chromeDir, "NativeMessagingHosts"), 0o700); err != nil {
		t.Fatal(err)
	}
	firefoxDir := filepath.Join(home, ".mozilla", "native-messaging-hosts")
	if err := os.MkdirAll(firefoxDir, 0o700); err != nil {
		t.Fatal(err)
	}
	l := &System{exe: mortar, home: home, configHome: cfg}

	okPath := filepath.Join(chromeDir, "NativeMessagingHosts", nativehost.Name+".json")
	okBody, err := nativehost.Manifest(mortar, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(okPath, okBody, 0o644); err != nil {
		t.Fatal(err)
	}

	stalePath := filepath.Join(firefoxDir, nativehost.Name+".json")
	staleBody, err := nativehost.Manifest(filepath.Join(home, "old-mortar"), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(stalePath, staleBody, 0o644); err != nil {
		t.Fatal(err)
	}

	rows := l.NativeHostStatus()
	byBrowser := map[string]HostStatus{}
	for _, row := range rows {
		byBrowser[row.Browser] = row
	}
	if byBrowser["Google Chrome"].State != HostOK {
		t.Fatalf("chrome: %+v", byBrowser["Google Chrome"])
	}
	if byBrowser["Firefox"].State != HostStale {
		t.Fatalf("firefox stale: %+v", byBrowser["Firefox"])
	}

	braveDir := filepath.Join(cfg, "BraveSoftware", "Brave-Browser")
	if err := os.MkdirAll(braveDir, 0o700); err != nil {
		t.Fatal(err)
	}
	rows = l.NativeHostStatus()
	for _, row := range rows {
		if row.Browser == "Brave" && row.State != HostMissing {
			t.Fatalf("brave missing: %+v", row)
		}
	}
}

func TestRepairNativeHostsWritesMissingManifest(t *testing.T) {
	l, _ := newLinux(t, "")
	vivaldi := filepath.Join(l.configHome, "vivaldi")
	if err := os.MkdirAll(vivaldi, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(vivaldi, "NativeMessagingHosts", nativehost.Name+".json")
	if err := l.WriteNativeHosts(); err != nil {
		t.Fatal(err)
	}
	b, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Path string `json:"path"`
	}
	if json.Unmarshal(b, &m) != nil || m.Path != l.exe {
		t.Fatalf("manifest: %s", b)
	}
}
