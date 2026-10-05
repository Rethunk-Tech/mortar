package profile

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fomod"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/installer"
)

// installerGame is the game as the installer sees it: the loaders it runs and the content targets the catalog gives it.
func installerGame(gameID string) installer.Game {
	info, _ := components.BundledGame(gameID)
	var g installer.Game
	for _, l := range info.Loaders {
		g.Loaders = append(g.Loaders, l.ID)
	}
	for _, t := range info.Targets {
		g.Targets = append(g.Targets, installer.Target{ID: t.ID, Root: t.Root, MaxDepth: t.MaxDepth})
	}
	return g
}

// layoutOf picks the installer for the store item key and lays it out: the archive (the item's chosen mod root), the
// files and where they go, and the driver's id. A FOMOD whose choices are missing or stale is a NeedChoicesError.
func (s *Store) layoutOf(game, id, key string, choices map[string]map[string][]string) (installer.Archive, installer.Layout, string, error) {
	root, err := s.items.Path(game, key)
	if err != nil {
		return installer.Archive{}, installer.Layout{}, "", err
	}
	arch := installer.Open(root, key)
	g := installerGame(game)
	inst, _ := installer.Pick(arch, g)
	if inst.ID() == "fomod" {
		cfg, _, _, err := fomod.Open(root)
		if err != nil {
			return arch, installer.Layout{}, "", err
		}
		modsDir, err := s.ModsDir(game, id)
		if err != nil {
			return arch, installer.Layout{}, "", err
		}
		arch.Eval = s.fomodEval(game, s.fileIndex(modsDir))
		if !fomod.Match(cfg, choices, arch.Eval) {
			return arch, installer.Layout{}, "", &NeedChoicesError{Ask: askFrom(cfg, key, Source{}, "", choices, arch.Eval)}
		}
	}
	l, err := inst.Layout(context.Background(), arch, g, choices)
	return arch, l, inst.ID(), err
}

// writeLayout puts the layout's files in dst as the item's own folder, so a file at <key>/a/b lands at dst/a/b.
func writeLayout(arch installer.Archive, l installer.Layout, dst string) error {
	for _, f := range l.Files {
		to := filepath.Join(dst, filepath.FromSlash(strings.TrimPrefix(f.Rel, arch.Key+"/")))
		if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
			return err
		}
		if err := datadir.MaterializeFile(filepath.Join(arch.Dir, filepath.FromSlash(f.Src)), to, f.Rel); err != nil {
			return err
		}
	}
	return nil
}

// layoutItem is a folder holding the store item as its entry lays out. A plain item is its own folder; any other is
// written to a temp folder, tmp, that the caller removes.
func (s *Store) layoutItem(game, id, key string, choices map[string]map[string][]string) (src, tmp string, err error) {
	arch, l, driver, err := s.layoutOf(game, id, key, choices)
	if err != nil || driver == "plain" {
		return arch.Dir, "", err
	}
	tmp, err = os.MkdirTemp("", "mortar-fomod-")
	if err != nil {
		return "", "", err
	}
	if err := writeLayout(arch, l, tmp); err != nil {
		_ = fsx.RemoveAll(tmp)
		return "", "", err
	}
	return tmp, tmp, nil
}
