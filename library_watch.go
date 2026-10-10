package main

import (
	"context"
	"log"
	"path/filepath"
	"slices"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/dlwatch"
	"github.com/Rethunk-Tech/mortar/internal/folderwatch"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/syncsvc"
)

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
// download folder until ctx ends.
func watchLibraryFolders(ctx context.Context, home, dataDir string, store *settings.Store, emit func(string, any)) {
	changed := make(chan struct{}, 1)
	store.OnChange(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	go func() {
		err := folderwatch.Run(ctx, folderwatch.Deps{
			Changed: changed,
			Emit: func(name string, data any) {
				log.Printf("folder watch: %s for %v", name, data)
				emit(name, data)
			},
			Targets: func() []folderwatch.Target {
				cur := store.Get()
				id := cur.LastGame
				if id == "" {
					ids := game.Implemented()
					if len(ids) != 1 {
						return nil
					}
					id = ids[0]
				}
				out := []folderwatch.Target{{Event: folderwatch.ExtraFolderEvent, Game: id, Dir: cur.GamePrefs(id).ExtraModsFolder}}
				for _, dir := range downloadDirs(store, dataDir) {
					out = append(out, folderwatch.Target{Event: folderwatch.DownloadsEvent, Game: id, Dir: dir})
				}
				if cur.SyncFolder != "" {
					for _, g := range game.Implemented() {
						out = append(out, folderwatch.Target{Event: syncsvc.FolderEvent, Game: g, Dir: filepath.Join(cur.SyncFolder, g)})
					}
				}
				if dir, err := game.InstallDir(home, cur, id); err == nil && dir != "" {
					out = append(out, folderwatch.Target{Event: folderwatch.ModsFolderEvent, Game: id, Dir: filepath.Join(dir, "Mods")})
				}
				return out
			},
		})
		if err != nil {
			log.Printf("folder watch: %v", err)
		}
	}()
}
