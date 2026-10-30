package nxm

import (
	"strconv"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/schemehandler"
	"golang.org/x/sys/windows/registry"
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

func readDefault(t *testing.T, key string) string {
	t.Helper()
	k, err := registry.OpenKey(registry.CURRENT_USER, key, registry.QUERY_VALUE)
	if err != nil {
		t.Fatalf("open %s: %v", key, err)
	}
	defer func() { _ = k.Close() }()
	v, _, err := k.GetStringValue("")
	if err != nil {
		t.Fatalf("read %s: %v", key, err)
	}
	return v
}

// A previous handler saved in settings by an earlier release, as "command\nicon\nname", still restores. The scheme is
// a throwaway one so no real handler is touched.
func TestReleaseRestoresAHandlerSavedByAnEarlierRelease(t *testing.T) {
	scheme := "mortartest" + strconv.FormatInt(time.Now().UnixNano(), 36)
	class := `Software\Classes\` + scheme
	t.Cleanup(func() { deleteTree(t, class) })
	h, err := schemehandler.New(schemehandler.Config{Exe: `C:\Mortar\mortar.exe`, Schemes: []string{scheme}})
	if err != nil {
		t.Fatal(err)
	}
	w := &System{exe: `C:\Mortar\mortar.exe`, h: h}
	if err := w.RegisterSchemes([]string{scheme}); err != nil {
		t.Fatal(err)
	}
	saved := `"C:\Vortex\Vortex.exe" -d "%1"` + "\n" + `C:\Vortex\Vortex.exe,0` + "\nURL:NXM Protocol"
	if err := w.Release([]string{scheme}, map[string]string{scheme: saved}); err != nil {
		t.Fatal(err)
	}
	if got := readDefault(t, class+`\shell\open\command`); got != `"C:\Vortex\Vortex.exe" -d "%1"` {
		t.Errorf("command %q", got)
	}
	if got := readDefault(t, class+`\DefaultIcon`); got != `C:\Vortex\Vortex.exe,0` {
		t.Errorf("icon %q", got)
	}
	if got := readDefault(t, class); got != "URL:NXM Protocol" {
		t.Errorf("name %q", got)
	}
	if o, err := w.Owner(scheme); err != nil || o.ID != saved || o.Name != `C:\Vortex\Vortex.exe` || o.Mine {
		t.Errorf("owner %+v, %v", o, err)
	}
}
