package queue

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/source/itch"
	"github.com/Rethunk-Tech/mortar/internal/source/modrinth"
)

// DirectFile is what a Modrinth or itch.io download needs: the file's address, size and digest, and the projects the
// version requires.
type DirectFile struct {
	ID, Name, Version, FileName, URL string
	SizeKB                           int64
	// Digest is "sha512:<hex>", or empty when the site published none.
	Digest       string
	Dependencies []DirectRef
}

// DirectRef names a required project; an empty Version means its newest.
type DirectRef struct{ ID, Version string }

// expandDirect resolves a Modrinth or itch.io request and its required dependencies, dependencies first. Each
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
			q.Kind, q.Disabled = KindDependency, nil
		}
		q.Package, q.Version, q.url, q.sizeKB, q.digest = ref.ID, f.Version, f.URL, f.SizeKB, f.Digest
		if !root || q.Name == "" {
			q.Name = cmp.Or(f.Name, ref.ID)
		}
		q.FileName = f.FileName
		out = append(out, q)
		return nil
	}
	if err := visit(DirectRef{ID: r.Package, Version: r.Version}, true, 0); err != nil {
		return nil, err
	}
	return out, nil
}

// sourceLoaders are the loaders the game's catalog lists for the source, which narrow the versions a dependency may take.
func sourceLoaders(gameID, sourceID string) []string {
	for _, g := range game.Catalog() {
		if g.ID != gameID {
			continue
		}
		if src, ok := g.Source(sourceID); ok {
			return src.Loaders
		}
	}
	return nil
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
	case "itch":
		r, err := itch.Driver{}.Resolve(ctx, id, version)
		if err != nil {
			return DirectFile{}, err
		}
		return DirectFile{ID: id, Version: r.UploadID, FileName: r.FileName, URL: r.URL, SizeKB: r.Size >> 10}, nil
	}
	return DirectFile{}, fmt.Errorf("unknown source %q", src)
}
