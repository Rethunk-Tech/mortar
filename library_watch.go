package main

import (
	"cmp"
	"context"
	"log/slog"
	"path/filepath"
	"slices"
	"sync"

	"github.com/Rethunk-Tech/go-folderwatch"
	"github.com/Rethunk-Tech/mortar/internal/devsvc"
	"github.com/Rethunk-Tech/mortar/internal/dlwatch"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/syncsvc"
)

const (
	ModsFolderEvent  = "library:mods-folder"
	ExtraFolderEvent = "library:extra-folder"
	DownloadsEvent   = "library:downloads"
)

// Folder is the extra-mods-folder and WithDir payload; json tags match the window's event.data.
type Folder struct {
	Game string `json:"game"`
	Dir  string `json:"dir"`
}

// downloadDirs are the folders new archives are offered from: Mortar's own download folder and the user's
// Downloads folder, which is looked up once because it runs a command.
func downloadDirs(store *settings.Store, dataDir string) []string {
	dirs := []string{archiveDir(store, dataDir), userDownloads()}
	for _, d := range store.Get().WatchedFolders() {
		if !slices.Contains(dirs, d) {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

var userDownloads = sync.OnceValue(dlwatch.UserDir)

func archiveDir(store *settings.Store, dataDir string) string {
	if d := store.Get().ArchiveDir(); filepath.IsAbs(d) {
		return d
	}
	return filepath.Join(dataDir, "downloads")
}

// watchLibraryFolders emits the library:* events for the open game's Mods folder, its extra mods folder and the
// download folder until ctx ends; holds takes folders out of the watch while they are moved.
func watchLibraryFolders(ctx context.Context, home, dataDir string, store *settings.Store, holds *folderwatch.Holds, emit func(string, any)) {
	changed := make(chan struct{}, 1)
	store.OnChange(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	walks := &devsvc.WalkCache{}
	go func() {
		err := folderwatch.Run(ctx, folderwatch.Deps{
			Changed: changed,
			Holds:   holds,
			Touched: walks.Touch,
			Emit: func(t folderwatch.Target) {
				slog.Warn("folder watch", "key", t.Key, "data", t.Data)
				emit(t.Key, t.Data)
			},
			Targets: func() []folderwatch.Target { return libraryWatchTargets(home, dataDir, store, walks) },
		})
		if err != nil {
			slog.Warn("folder watch", "err", err)
		}
	}()
}

// libraryWatchTargets lists the folders to watch for the settings as they are now.
func libraryWatchTargets(home, dataDir string, store *settings.Store, walks *devsvc.WalkCache) []folderwatch.Target {
	cur := store.Get()
	id := cur.LastGame
	if id == "" {
		ids := game.Implemented()
		if len(ids) != 1 {
			return nil
		}
		id = ids[0]
	}
	var out []folderwatch.Target
	for _, dir := range downloadDirs(store, dataDir) {
		out = append(out, folderwatch.Target{Key: DownloadsEvent, Dir: dir, Data: id, Settle: true})
	}
	for _, g := range game.Implemented() {
		for _, dir := range cur.GamePrefs(g).ExtraModsFolderList() {
			out = append(out, folderwatch.Target{Key: ExtraFolderEvent, Dir: dir, Data: Folder{Game: g, Dir: dir}})
		}
		if dir := cur.SyncFolderFor(g); dir != "" {
			out = append(out, folderwatch.Target{Key: syncsvc.FolderEvent, Dir: filepath.Join(dir, g), Data: g})
		}
	}
	for _, f := range devsvc.AssetSubfolders(cur, game.Implemented(), walks) {
		out = append(out, folderwatch.Target{Key: ExtraFolderEvent, Dir: f.Dir, Report: f.Root, Data: Folder{Game: f.Game, Dir: cmp.Or(f.Root, f.Dir)}})
	}
	for g, dirs := range devsvc.WatchedSources(cur, game.Implemented(), walks) {
		for _, dir := range dirs {
			out = append(out, folderwatch.Target{Key: devsvc.SourceEvent, Dir: dir, Data: Folder{Game: g, Dir: dir}})
		}
	}
	if rel, ok := game.ManifestModsFolder(id); ok {
		if dir, err := game.InstallDir(home, cur, id); err == nil && dir != "" {
			out = append(out, folderwatch.Target{Key: ModsFolderEvent, Dir: filepath.Join(dir, rel), Data: id})
		}
	}
	return out
}
