package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-folderwatch"
	"github.com/Rethunk-Tech/mortar/internal/devsvc"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestLibraryWatchTargetsPinEventNamesAndPayloads(t *testing.T) {
	home := t.TempDir()
	dataDir := t.TempDir()
	extra := t.TempDir()
	src := t.TempDir()
	install := t.TempDir()
	if err := os.WriteFile(filepath.Join(install, "Stardew Valley.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	dl := filepath.Join(dataDir, "downloads")
	orig := userDownloads
	t.Cleanup(func() { userDownloads = orig })
	userDownloads = func() string { return dl }

	store, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(func(v *settings.Settings) {
		v.LastGame = "stardew"
		if v.GameFolders == nil {
			v.GameFolders = map[string]string{}
		}
		v.GameFolders["stardew"] = install
		gp := v.GamePrefs("stardew")
		gp.ExtraModsFolder = extra
		gp.DevProjects = []settings.DevProject{{Dir: src, Name: "Mod", Watch: true}}
		if v.Games == nil {
			v.Games = map[string]*settings.GameSettings{}
		}
		v.Games["stardew"] = &gp
	}); err != nil {
		t.Fatal(err)
	}

	got := libraryWatchTargets(home, dataDir, store, &devsvc.WalkCache{})
	check := func(kind string, tg folderwatch.Target, key string, data any, settle bool) {
		if tg.Key != key {
			t.Fatalf("%s key %q, want %q", kind, tg.Key, key)
		}
		want, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}
		have, err := json.Marshal(tg.Data)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(have, want) {
			t.Fatalf("%s payload %s, want %s", kind, have, want)
		}
		if tg.Settle != settle {
			t.Fatalf("%s settle %v, want %v", kind, tg.Settle, settle)
		}
	}

	var downloads, extraT, mods, source folderwatch.Target
	var sawDownloads, sawExtra, sawMods, sawSource bool
	assets := filepath.Join(src, "assets")
	for _, tg := range got {
		switch {
		case tg.Key == DownloadsEvent && tg.Dir == dl:
			downloads, sawDownloads = tg, true
		case tg.Key == ExtraFolderEvent && tg.Dir == extra:
			extraT, sawExtra = tg, true
		case tg.Key == ModsFolderEvent:
			mods, sawMods = tg, true
		case tg.Key == devsvc.SourceEvent && tg.Dir == assets:
			source, sawSource = tg, true
		}
	}
	if !sawDownloads || !sawExtra || !sawMods || !sawSource {
		t.Fatalf("missing kinds downloads=%v extra=%v mods=%v source=%v in %+v", sawDownloads, sawExtra, sawMods, sawSource, got)
	}
	check("downloads", downloads, DownloadsEvent, "stardew", true)
	check("extra-folder", extraT, ExtraFolderEvent, Folder{Game: "stardew", Dir: extra}, false)
	check("mods-folder", mods, ModsFolderEvent, "stardew", false)
	check("source", source, devsvc.SourceEvent, Folder{Game: "stardew", Dir: assets}, false)
}
