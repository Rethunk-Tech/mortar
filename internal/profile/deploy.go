package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deploy"
	"github.com/Rethunk-Tech/mortar/internal/installer"
)

// OverwriteDir is where a launch keeps the files the game wrote outside its writable targets: <profile>/overwrite.
const OverwriteDir = "overwrite"

// DeployInputs is what a launch hands the deployer for a profile of a game that deploys into its install.
type DeployInputs struct {
	// Packages are lowest priority first: the profile's enabled entries in profile order, then the profile's own
	// writable target folders (its saved configs), then the overwrite folder.
	Packages []deploy.Package
	Targets  []deploy.Target
	// Overwrite is the folder harvest sends new files to.
	Overwrite string
}

// profileRoot resolves a catalog root token below the profile folder.
func profileRoot(dir, root string) string {
	return filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(strings.TrimPrefix(root, "{profile}"), "/")))
}

// DeployInputs lays out every enabled entry of the profile for the game's install-side targets. installDir is the
// install folder, which the catalog's {install} token stands for. A game that is redirected gets none.
func (s *Store) DeployInputs(gameID, id, installDir string) (DeployInputs, error) {
	info, ok := components.BundledGame(gameID)
	if !ok || info.Deploy != components.DeployLink {
		return DeployInputs{}, nil
	}
	p, dir, err := s.readDir(gameID, id)
	if err != nil {
		return DeployInputs{}, err
	}
	in := DeployInputs{Overwrite: filepath.Join(dir, OverwriteDir)}
	for _, t := range info.Targets {
		root := filepath.Join(installDir, filepath.FromSlash(strings.TrimPrefix(strings.TrimPrefix(t.Install, "{install}"), "/")))
		dt := deploy.Target{ID: t.ID, Root: root, Writable: t.Writable}
		if t.Writable {
			dt.Home = profileRoot(dir, t.Root)
		}
		in.Targets = append(in.Targets, dt)
	}
	for _, e := range p.Entries {
		if e.IsOverlay() || !e.hasPackageEnabled() {
			continue
		}
		arch, l, _, err := s.layoutOf(gameID, id, e.Key, e.Fomod)
		if err != nil {
			return DeployInputs{}, fmt.Errorf("%s: %w", entryLabel(e), err)
		}
		in.Packages = append(in.Packages, deploy.Package{ID: e.Key, Root: arch.Dir, Layout: l})
	}
	for _, t := range info.Targets {
		if !t.Writable {
			continue
		}
		pkg, err := folderPackage("profile-"+t.ID, profileRoot(dir, t.Root), t.ID)
		if err != nil {
			return DeployInputs{}, err
		}
		in.Packages = append(in.Packages, pkg)
	}
	over := deploy.Package{ID: "overwrite", Root: in.Overwrite}
	for _, t := range info.Targets {
		pkg, err := folderPackage("overwrite", filepath.Join(in.Overwrite, t.ID), t.ID)
		if err != nil {
			return DeployInputs{}, err
		}
		for _, f := range pkg.Layout.Files {
			f.Src = t.ID + "/" + f.Src
			over.Layout.Files = append(over.Layout.Files, f)
		}
	}
	in.Packages = append(in.Packages, over)
	return in, nil
}

// hasPackageEnabled reports an entry with no mods to switch off, or with at least one switched on.
func (e Entry) hasPackageEnabled() bool {
	return len(e.Mods) == 0 || slices.ContainsFunc(e.Mods, func(m Component) bool { return e.Enabled(m.ID) })
}

// folderPackage is every file below root as a package of target.
func folderPackage(id, root, target string) (deploy.Package, error) {
	pkg := deploy.Package{ID: id, Root: root}
	err := fs.WalkDir(os.DirFS(root), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !d.Type().IsRegular() {
			return err
		}
		pkg.Layout.Files = append(pkg.Layout.Files, installer.File{Src: p, Target: target, Rel: p})
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return deploy.Package{}, err
	}
	return pkg, nil
}
