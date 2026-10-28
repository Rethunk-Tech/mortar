package savessvc

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/saves"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestRestoreSaveCopyRefusedWhileBusy(t *testing.T) {
	s, savesDir := backupService(t)
	s.busy = func() bool { return true }
	testfs.WriteFarm(t, savesDir, "Farm_1", "Sunny", "current")
	if err := fsx.WriteFile(filepath.Join(savesDir, "Farm_1", "Farm_1_old"), []byte("older"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreSaveCopy(t.Context(), "stardew", "", "Farm_1", "Farm_1_old"); !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
	got, err := fsx.ReadFile(filepath.Join(savesDir, "Farm_1", "Farm_1"))
	if err != nil || string(got) != "current" {
		t.Fatalf("save after a refused restore: %q %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(savesDir, "Farm_1", "Farm_1_old")); err != nil {
		t.Fatalf("copy gone: %v", err)
	}
}

func TestFitForDismissedModIsNotMissing(t *testing.T) {
	t.Parallel()
	in := saves.Info{Folder: "Farm_1", Used: []mod.ID{"smapi:a.mod", "smapi:b.mod"}}
	have := map[string]bool{"smapi:b.mod": false}
	for name, tc := range map[string]struct {
		dismissed []mod.ID
		want      []Lack
		ok        bool
	}{
		"nothing dismissed": {nil, []Lack{{ID: "smapi:a.mod", Name: "a.mod"}, {ID: "smapi:b.mod", Name: "b.mod", Disabled: true}}, true},
		"one dismissed":     {[]mod.ID{"smapi:A.Mod"}, []Lack{{ID: "smapi:b.mod", Name: "b.mod", Disabled: true}}, true},
		"all dismissed":     {[]mod.ID{"smapi:a.mod", "smapi:b.mod"}, []Lack{}, false},
	} {
		fit, ok := fitFor(in, have, tc.dismissed)
		if ok != tc.ok || len(fit.Missing) != len(tc.want) {
			t.Errorf("%s: ok=%v missing=%+v", name, ok, fit.Missing)
			continue
		}
		for i, l := range tc.want {
			if fit.Missing[i] != l {
				t.Errorf("%s: missing[%d] = %+v, want %+v", name, i, fit.Missing[i], l)
			}
		}
	}
}

func TestLastSaveGap(t *testing.T) {
	e := newSaveEnv(t)
	p, err := e.profiles.Create("stardew", "Main")
	if err != nil {
		t.Fatal(err)
	}
	e.alphaItem(t)
	if _, err := e.profiles.AddEntry("stardew", p.ID, "local-a", profile.Source{Kind: profile.KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	index := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"A.Mod": ["Nexus:5"]}`)) }))
	t.Cleanup(index.Close)
	st, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	savesDir := t.TempDir()
	s := &Service{
		profiles: e.profiles, settings: st, last: NewStore(t.TempDir()),
		meta:     &meta.Client{HTTP: index.Client(), CacheDir: t.TempDir(), IndexURL: index.URL},
		scanners: map[string]*saves.Scanner{"stardew": {Dir: savesDir, CacheDir: t.TempDir()}},
	}

	if _, ok, err := s.LastSaveGap(t.Context(), "stardew", p.ID); ok || err != nil {
		t.Fatalf("no saves: ok=%v err=%v", ok, err)
	}

	body := "<SaveGame><modData><item><key><string>A.Mod/x</string></key><value><string>1</string></value></item></modData></SaveGame>"
	testfs.WriteAt(t, filepath.Join(savesDir, "Farm_1", "Farm_1"), []byte(body))
	if _, ok, err := s.LastSaveGap(t.Context(), "stardew", p.ID); ok || err != nil {
		t.Fatalf("every mod enabled: ok=%v err=%v", ok, err)
	}

	if _, err := e.profiles.SetModEnabled("stardew", p.ID, "local-a", "smapi:A.Mod", false); err != nil {
		t.Fatal(err)
	}
	fit, ok, err := s.LastSaveGap(t.Context(), "stardew", p.ID)
	if err != nil || !ok || fit.Folder != "Farm_1" || len(fit.Missing) != 1 || !fit.Missing[0].Disabled {
		t.Fatalf("mod switched off: %+v ok=%v err=%v", fit, ok, err)
	}
}
