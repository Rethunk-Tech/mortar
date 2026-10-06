package queue

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/source/curseforge"
	"github.com/Rethunk-Tech/mortar/internal/source/itch"
	"github.com/Rethunk-Tech/mortar/internal/source/modrinth"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// DirectFile is what a Modrinth, CurseForge or itch.io download needs: the file's address, size and digest, and the projects the
// version requires.
type DirectFile struct {
	ID, Name, Version, FileName, URL string
	SizeKB                           int64
	// FileID is the site's exact file id where it has one (CurseForge), 0 otherwise.
	FileID int
	// Digest is "sha512:<hex>", or empty when the site published none.
	Digest       string
	Dependencies []DirectRef
}

// DirectRef names a required project; an empty Version means its newest.
type DirectRef struct{ ID, Version string }

// expandDirect resolves a Modrinth, CurseForge or itch.io request and its required dependencies, dependencies first. Each
// project appears once.
func (s *Service) expandDirect(ctx context.Context, r Request) ([]Request, error) {
	if s.d.Direct == nil {
		return nil, errors.New("cannot install " + r.Source + " mods here")
	}
	loaders := sourceLoaders(r.Game, r.Source)
	var out []Request
	seen := map[string]bool{}
	var visit func(ref DirectRef, root bool, depth int) error
	visit = func(ref DirectRef, root bool, depth int) error {
		key := strings.ToLower(ref.ID)
		if seen[key] {
			return nil
		}
		seen[key] = true
		if depth > 16 {
			return fmt.Errorf("%s dependencies nest too deep at %s", r.Source, ref.ID)
		}
		cctx, cancel := context.WithTimeout(ctx, closureTimeout)
		f, err := s.d.Direct(cctx, r.Source, ref.ID, ref.Version, loaders)
		cancel()
		if manual, ok := errors.AsType[*curseforge.NotDistributableError](err); ok {
			return s.handOff(manual)
		}
		if err != nil {
			return err
		}
		for _, d := range f.Dependencies {
			if err := visit(d, false, depth+1); err != nil {
				return err
			}
		}
		q := r
		if !root {
			q.Kind, q.Disabled, q.Picture = KindDependency, nil, ""
		}
		q.Package, q.Version, q.url, q.sizeKB, q.digest = ref.ID, f.Version, f.URL, f.SizeKB, f.Digest
		if !root || q.Name == "" {
			q.Name = cmp.Or(f.Name, ref.ID)
		}
		q.FileName = f.FileName
		q.PackageFile = f.FileID
		out = append(out, q)
		return nil
	}
	rootVersion := r.Version
	if r.PackageFile != 0 {
		rootVersion = strconv.Itoa(r.PackageFile)
	}
	if err := visit(DirectRef{ID: r.Package, Version: rootVersion}, true, 0); err != nil {
		return nil, err
	}
	return out, nil
}

// handOff opens the page of a mod whose author forbids downloads outside CurseForge, the way a Nexus file Mortar may
// not fetch is left to the site, and fails the request with what the user does next.
func (s *Service) handOff(e *curseforge.NotDistributableError) error {
	msg := e.Error() + ": download the file from its CurseForge page and add it from your computer"
	if s.d.OpenURL == nil {
		return usererr.New(usererr.External, msg)
	}
	if err := s.d.OpenURL(e.PageURL); err != nil {
		return err
	}
	return usererr.New(usererr.External, msg+" (the page is open)")
}

// sourceLoaders are the loaders the game's catalog lists for the source, which narrow the versions a dependency may take.
func sourceLoaders(gameID, sourceID string) []string {
	if g, ok := gameInfo(gameID); ok {
		if src, ok := g.Source(sourceID); ok {
			return src.Loaders
		}
	}
	return nil
}

func gameInfo(gameID string) (components.GameInfo, bool) {
	for _, g := range game.Catalog() {
		if g.ID == gameID {
			return g, true
		}
	}
	return components.GameInfo{}, false
}

// ResolveDirect is Deps.Direct for the real sites.
func ResolveDirect(ctx context.Context, src, id, version string, loaders []string) (DirectFile, error) {
	switch src {
	case "modrinth":
		r, err := modrinth.Driver{}.Resolve(ctx, id, version, "", loaders)
		if err != nil {
			return DirectFile{}, err
		}
		f := DirectFile{ID: id, Version: r.Version, FileName: r.FileName, URL: r.URL, SizeKB: r.Size >> 10, Digest: r.Digest}
		for _, d := range r.Dependencies {
			f.Dependencies = append(f.Dependencies, DirectRef{ID: d.ProjectID, Version: d.VersionID})
		}
		return f, nil
	case "curseforge":
		r, err := curseforge.Driver{}.Resolve(ctx, id, version)
		if err != nil {
			return DirectFile{}, err
		}
		fileID, _ := strconv.Atoi(r.VersionID)
		f := DirectFile{ID: id, Name: r.Name, Version: r.Version, FileID: fileID, FileName: r.FileName, URL: r.URL, SizeKB: r.Size >> 10, Digest: r.Digest}
		for _, d := range r.Dependencies {
			f.Dependencies = append(f.Dependencies, DirectRef{ID: d})
		}
		return f, nil
	case "itch":
		r, err := itch.Driver{}.Resolve(ctx, id, version)
		if err != nil {
			return DirectFile{}, err
		}
		return DirectFile{ID: id, Version: r.UploadID, FileName: r.FileName, URL: r.URL, SizeKB: r.Size >> 10}, nil
	}
	return DirectFile{}, fmt.Errorf("unknown source %q", src)
}
