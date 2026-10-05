package queue

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

// closureTimeout bounds the community index refresh a package request may wait on.
const closureTimeout = 2 * time.Minute

// expandPackages replaces each Thunderstore request with the package and everything it depends on, dependencies
// first, each with its download address; the other requests pass through.
func (s *Service) expandPackages(ctx context.Context, reqs []Request) ([]Request, error) {
	var out []Request
	for _, r := range reqs {
		if r.Package == "" {
			out = append(out, r)
			continue
		}
		if s.d.Closure == nil {
			return nil, errors.New("cannot install Thunderstore packages here")
		}
		ns, name, _ := strings.Cut(r.Package, "-")
		ctx, cancel := context.WithTimeout(ctx, closureTimeout)
		list, err := s.d.Closure(ctx, r.Game, []thunderstore.Ref{{Namespace: ns, Name: name, Version: r.Version}})
		cancel()
		if err != nil {
			return nil, err
		}
		for _, p := range list {
			id := p.Namespace + "-" + p.Name
			dep := r
			dep.Package, dep.Name, dep.Version, dep.url, dep.sizeKB = id, p.Name, p.Version, p.URL, p.Size>>10
			dep.FileName = id + "-" + p.Version + ".zip"
			if !strings.EqualFold(id, r.Package) {
				dep.Kind = KindDependency
			}
			out = append(out, dep)
		}
	}
	return out, nil
}

func packageSource(it Item) profile.Source {
	return profile.Source{Kind: profile.KindThunderstore, Name: it.Package, Version: it.Version}
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
	if err := s.checkPackageHash(it, path); err != nil {
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
