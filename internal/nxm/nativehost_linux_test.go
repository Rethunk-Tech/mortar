package nxm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/nativehost"
)

func TestRegisterWritesHostManifestsForInstalledBrowsersAndRestoreRemovesThem(t *testing.T) {
	l, _ := newLinux(t, "")
	vivaldi := filepath.Join(l.configHome, "vivaldi")
	for _, d := range []string{vivaldi, filepath.Join(l.home, ".mozilla")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	chromium := filepath.Join(vivaldi, "NativeMessagingHosts", nativehost.Name+".json")
	firefox := filepath.Join(l.home, ".mozilla", "native-messaging-hosts", nativehost.Name+".json")
	var m struct {
		Path              string   `json:"path"`
		AllowedOrigins    []string `json:"allowed_origins"`
		AllowedExtensions []string `json:"allowed_extensions"`
	}
	b, err := fsx.ReadFile(chromium)
	if err != nil || json.Unmarshal(b, &m) != nil || m.Path != l.exe || len(m.AllowedOrigins) != 1 || m.AllowedOrigins[0] != nativehost.ChromeOrigin {
		t.Fatalf("chromium manifest %s, %v", b, err)
	}
	m.AllowedOrigins = nil
	b, err = fsx.ReadFile(firefox)
	if err != nil || json.Unmarshal(b, &m) != nil || m.AllowedOrigins != nil || len(m.AllowedExtensions) != 1 {
		t.Fatalf("firefox manifest %s, %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(l.configHome, "google-chrome")); err == nil {
		t.Error("created a folder for a browser that is not installed")
	}
	if err := l.Restore(""); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{chromium, firefox} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s survived Restore: %v", p, err)
		}
	}
}
