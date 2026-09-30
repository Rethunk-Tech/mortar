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
)

// release finds the stable release an item names and its archive assets.
func (s *Service) release(ctx context.Context, it Item) (github.Release, []github.Asset, error) {
	owner, repo, _ := strings.Cut(it.Repo, "/")
	all, err := s.d.GitHub.Releases(ctx, owner, repo)
	if err != nil {
		return github.Release{}, nil, err
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
	return profile.Source{
		Kind: profile.KindGitHub, Name: it.Asset, Version: it.Version, Repo: it.Repo, Tag: it.Tag, Asset: it.Asset,
	}
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
	p := &progress{s: s, id: it.ID, total: assets[i].Size, last: s.d.Now()}
	path, err := s.d.GitHub.Download(ctx, assets[i], func(done, total int64) {
		if total > 0 {
			p.total = total
		}
		p.set(done)
	})
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return s.diskError(err, assets[i].Size)
	}
	defer func() { _ = os.Remove(path) }()
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
	_, err = s.d.InstallStaged(it.Game, it.Profile, key, source)
	return s.finish(it.ID, err, unverified)
}

// installStaged installs what Confirm released. The user has decided, so nothing is verified again.
func (s *Service) installStaged(it Item) error {
	_, err := s.d.InstallStaged(it.Game, it.Profile, it.staged, sourceOf(it))
	return s.finish(it.ID, err, false)
}
