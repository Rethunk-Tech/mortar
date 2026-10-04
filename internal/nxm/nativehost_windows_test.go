package nxm

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/windows/registry"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func deleteTree(t *testing.T, path string) {
	t.Helper()
	k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return
	}
	subs, _ := k.ReadSubKeyNames(-1)
	_ = k.Close()
	for _, s := range subs {
		deleteTree(t, path+`\`+s)
	}
	if err := registry.DeleteKey(registry.CURRENT_USER, path); err != nil {
		t.Errorf("delete %s: %v", path, err)
	}
}

func TestWriteNativeHostsPointsEveryBrowserAtItsManifestAndRemoveUndoesIt(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	root := `Software\MortarTest` + strconv.FormatInt(time.Now().UnixNano(), 36)
	t.Cleanup(func() { deleteTree(t, root) })
	w := &System{exe: `C:\Mortar\mortar.exe`, software: root}
	if err := w.WriteNativeHosts(); err != nil {
		t.Fatal(err)
	}
	for key, firefox := range hostKeys(root) {
		k, err := registry.OpenKey(registry.CURRENT_USER, key, registry.QUERY_VALUE)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		path, _, err := k.GetStringValue("")
		_ = k.Close()
		if err != nil {
			t.Fatal(err)
		}
		var m struct {
			Path              string   `json:"path"`
			AllowedOrigins    []string `json:"allowed_origins"`
			AllowedExtensions []string `json:"allowed_extensions"`
		}
		b, err := fsx.ReadFile(path)
		if err != nil || json.Unmarshal(b, &m) != nil || m.Path != w.exe {
			t.Fatalf("%s -> %s: %s, %v", key, path, b, err)
		}
		if firefox != (len(m.AllowedExtensions) == 1 && m.AllowedOrigins == nil) {
			t.Errorf("%s has the wrong allow-list: %s", key, b)
		}
	}
	if err := w.removeNativeHosts(); err != nil {
		t.Fatal(err)
	}
	for key := range hostKeys(root) {
		if k, err := registry.OpenKey(registry.CURRENT_USER, key, registry.QUERY_VALUE); err == nil {
			_ = k.Close()
			t.Errorf("%s survived removal", key)
		}
	}
	dir, _ := hostManifestDir()
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("manifests survived removal in %s", dir)
	}
}
