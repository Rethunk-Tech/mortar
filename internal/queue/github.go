package queue

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/github"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/store"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

// release finds the stable release an item names and its archive assets.
func (s *Service) release(ctx context.Context, it Item) (github.Release, []github.Asset, error) {
	owner, repo, _ := strings.Cut(it.Repo, "/")
	all, err := s.d.GitHub.Releases(ctx, owner, repo)
	if err != nil {
		return github.Release{}, nil, usererr.Wrap(usererr.Network, err)
	}
	rel, assets, err := github.Select(all, cmp.Or(it.Tag, it.Version))
	if err != nil {
		return github.Release{}, nil, fmt.Errorf("%s: %w", it.Repo, err)
	}
	return rel, assets, nil
}

// resolveGitHub fixes the item's release and asset, or hands the choice to the user when the release has several
// archives.
func (s *Service) resolveGitHub(ctx context.Context, it Item) error {
	rel, assets, err := s.release(ctx, it)
	if err != nil {
		return err
	}
	if it.Asset != "" && !slices.ContainsFunc(assets, func(a github.Asset) bool { return a.Name == it.Asset }) {
		return fmt.Errorf("release %s of %s has no asset %s", rel.Tag, it.Repo, it.Asset)
	}
	s.mu.Lock()
	if cur := s.find(it.ID); cur != nil {
		cur.Tag = rel.Tag
		switch {
		case cur.Asset != "":
		case len(assets) == 1:
			cur.Asset = assets[0].Name
		default:
			cur.State = StateNeedsChoice
			cur.Assets = make([]string, len(assets))
			for i, a := range assets {
				cur.Assets[i] = a.Name
			}
		}
		cur.FileName = cur.Asset
	}
	s.mu.Unlock()
	s.publish(true)
	return nil
}

func sourceOf(it Item) profile.Source {
	return sourceWithOptions(it, profile.Source{
		Kind: profile.KindGitHub, Name: it.Asset, Version: it.Version, Repo: it.Repo, Tag: it.Tag, Asset: it.Asset,
	})
}

// downloadGitHub fetches the item's asset, stages it, and installs it when SMAPI's API ties its mods to the repo.
func (s *Service) downloadGitHub(ctx context.Context, it Item) error {
	_, assets, err := s.release(ctx, it)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(assets, func(a github.Asset) bool { return a.Name == it.Asset })
	if i < 0 {
		return fmt.Errorf("release %s of %s has no asset %s", it.Tag, it.Repo, it.Asset)
	}
	if err := os.MkdirAll(s.downloadRoot(), 0o700); err != nil {
		return err
	}
	path := s.dest(it.ID, it.Asset)
	if err := s.fetch(ctx, it, assets[i].URL, path); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
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
	source := sourceOf(it)
	key, ids, err := s.d.Stage(it.Game, source, path)
	if err != nil {
		return err
	}
	if s.d.KeepArchives == nil || !s.d.KeepArchives() {
		dropDownload(path)
	}
	owner, repo, _ := strings.Cut(it.Repo, "/")
	unverified := false
	for _, id := range ids {
		ok, verr := s.d.Verify(ctx, id, owner, repo)
		switch {
		case errors.Is(verr, github.ErrUnknown):
			unverified = true
		case verr != nil:
			return verr
		case !ok:
			s.mu.Lock()
			if cur := s.find(it.ID); cur != nil {
				cur.State, cur.Progress, cur.staged = StateNeedsConfirm, 0, key
			}
			s.mu.Unlock()
			s.publish(true)
			return nil
		}
	}
	s.installMu.Lock()
	res, err := s.d.InstallStaged(it.Game, it.Profile, key, source)
	s.installMu.Unlock()
	return s.afterInstall(it.ID, res, err, unverified)
}

// installStaged installs what Confirm released. The user has decided, so nothing is verified again.
// A staged key the store no longer holds is dropped, so Retry downloads the file again.
func (s *Service) installStaged(ctx context.Context, it Item) error {
	if ok, err := s.installReadyZip(ctx, it); ok || err != nil {
		return s.contentPatcherHint(ctx, it, err)
	}
	src := sourceOf(it)
	if it.chosenRoot != "" {
		if it.Remap != nil {
			src = it.Remap.Source
		}
		s.installMu.Lock()
		res, err := s.d.InstallRemap(it.Game, it.Profile, it.staged, it.chosenRoot, src)
		s.installMu.Unlock()
		if errors.Is(err, store.ErrNotFound) {
			s.mu.Lock()
			if cur := s.find(it.ID); cur != nil {
				cur.staged, cur.chosenRoot = "", ""
			}
			s.mu.Unlock()
		}
		return s.afterInstall(it.ID, res, err, false)
	}
	s.installMu.Lock()
	res, err := s.d.InstallStaged(it.Game, it.Profile, it.staged, src)
	s.installMu.Unlock()
	if errors.Is(err, store.ErrNotFound) {
		s.mu.Lock()
		if cur := s.find(it.ID); cur != nil {
			cur.staged = ""
		}
		s.mu.Unlock()
	}
	return s.afterInstall(it.ID, res, err, false)
}
