package queue

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

// closureTimeout bounds the community index refresh a package request may wait on.
const closureTimeout = 2 * time.Minute

// loaderPackage is the BepInEx pack. The profile's loader provides it, so it is never queued as a package.
const loaderPackage = "bepinex-bepinexpack"

// expandPackages replaces the Thunderstore requests of each game and profile with their packages and everything they
// depend on, dependencies first, each with its download address; the other requests pass through. One closure covers
// all the roots of a group, so a dependency shared by two roots resolves once, at the highest version either asks
// for, and one the profile already holds at that version or newer is not queued again.
func (s *Service) expandPackages(ctx context.Context, reqs []Request) ([]Request, error) {
	var out []Request
	done := map[int]bool{}
	for i, r := range reqs {
		if r.Package == "" {
			out = append(out, r)
			continue
		}
		if r.Source != "" {
			direct, err := s.expandDirect(ctx, r)
			if err != nil {
				return nil, err
			}
			out = append(out, direct...)
			continue
		}
		if done[i] {
			continue
		}
		if s.d.Closure == nil {
			return nil, errors.New("cannot install Thunderstore packages here")
		}
		var roots []thunderstore.Ref
		byID := map[string]Request{}
		for j := i; j < len(reqs); j++ {
			q := reqs[j]
			if q.Package == "" || q.Source != "" || q.Game != r.Game || q.Profile != r.Profile {
				continue
			}
			done[j] = true
			ns, name, _ := strings.Cut(q.Package, "-")
			roots = append(roots, thunderstore.Ref{Namespace: ns, Name: name, Version: q.Version})
			byID[strings.ToLower(q.Package)] = q
		}
		cctx, cancel := context.WithTimeout(ctx, closureTimeout)
		list, err := s.d.Closure(cctx, r.Game, roots)
		cancel()
		if err != nil {
			return nil, err
		}
		for _, p := range list {
			id := p.Namespace + "-" + p.Name
			low := strings.ToLower(id)
			if low == loaderPackage {
				continue
			}
			dep, isRoot := byID[low]
			if !isRoot {
				if s.holds(r, id, p.Version) {
					continue
				}
				dep = r
				dep.Kind, dep.Disabled = KindDependency, nil
			}
			dep.Package, dep.Name, dep.Version, dep.url, dep.sizeKB = id, p.Name, p.Version, p.URL, p.Size>>10
			dep.picture, dep.category = p.Icon, p.Category
			dep.FileName = id + "-" + p.Version + ".zip"
			out = append(out, dep)
		}
	}
	return out, nil
}

// holds reports whether the request's profile already has the package at version or newer.
func (s *Service) holds(r Request, id, version string) bool {
	if s.d.Held == nil {
		return false
	}
	have := s.d.Held(r.Game, r.Profile, id)
	return have != "" && !meta.Newer(version, have)
}

func packageSource(it Item) profile.Source {
	src := profile.Source{
		Kind: cmp.Or(it.Source, profile.KindThunderstore), Name: it.Package, Version: it.Version, Digest: it.Digest,
		Picture: it.Picture, Category: it.Category,
	}
	if len(it.Disabled) > 0 {
		src = src.WithDisabled(it.Disabled)
	}
	return src
}

// downloadPackage fetches a package archive and installs it into the item's profile.
func (s *Service) downloadPackage(ctx context.Context, it Item) error {
	if s.d.InstallPackage == nil {
		return errors.New("cannot install Thunderstore packages here")
	}
	if it.URL == "" {
		return errors.New("the package has no download address: add it again")
	}
	if err := os.MkdirAll(s.downloadRoot(), 0o700); err != nil {
		return err
	}
	path := s.dest(it.ID, it.FileName)
	if err := s.fetch(ctx, it, it.URL, path); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%s: %w", it.Package, err)
	}
	if err := s.checkPackage(it, path); err != nil {
		dropDownload(path)
		return err
	}
	s.mu.Lock()
	cur := s.find(it.ID)
	if cur == nil || cur.State != StateDownloading {
		s.mu.Unlock()
		return context.Canceled
	}
	cur.State, cur.Progress, cur.Speed = StateInstalling, 100, 0
	s.mu.Unlock()
	s.publish(true)
	s.installMu.Lock()
	res, err := s.d.InstallPackage(it.Game, it.Profile, path, packageSource(it))
	s.installMu.Unlock()
	var dup *profile.DuplicateError
	if err == nil || errors.As(err, &dup) {
		s.dropDownloadUnlessKept(path)
	}
	return s.afterInstall(it.ID, res, err, false)
}
