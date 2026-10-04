package main

import (
	"context"
	"log"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/folderwatch"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

// archiveDir is where the queue's archives and hand-placed ones live.
func archiveDir(store *settings.Store, dataDir string) string {
	if d := store.Get().ArchiveDir(); filepath.IsAbs(d) {
		return d
	}
	return filepath.Join(dataDir, "downloads")
}

// watchLibraryFolders emits the library:* events for the open game's Mods folder, its extra mods folder and the
// download folder until ctx ends.
func watchLibraryFolders(ctx context.Context, home, dataDir string, store *settings.Store, emit func(string, any)) {
	go func() {
		err := folderwatch.Run(ctx, folderwatch.Deps{
			Emit: emit,
			Targets: func() []folderwatch.Target {
				cur := store.Get()
				id := cur.LastGame
				if id == "" {
					id = "stardew"
				}
				out := []folderwatch.Target{
					{Event: folderwatch.ExtraFolderEvent, Game: id, Dir: cur.GamePrefs(id).ExtraModsFolder},
					{Event: folderwatch.DownloadsEvent, Game: id, Dir: archiveDir(store, dataDir)},
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
