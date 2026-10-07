package queue

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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

// choicesFile maps a repository ("owner/repo", lower case) to the Shape of the asset the user last chose from it.
const choicesFile = "github-choices.json"

func (s *Service) choices() map[string]string {
	m := map[string]string{}
	_, _ = datadir.ReadJSON(filepath.Join(s.d.Dir, choicesFile), &m)
	return m
}

// remember keeps the shape of the asset the user chose from repo, so the next release picks the same download.
func (s *Service) remember(repo, asset string) {
	s.choiceMu.Lock()
	defer s.choiceMu.Unlock()
	m := s.choices()
	m[strings.ToLower(repo)] = github.Shape(asset)
	if err := datadir.WriteJSON(filepath.Join(s.d.Dir, choicesFile), m); err != nil {
		log.Printf("queue: remember the %s asset choice: %v", repo, err)
	}
}

// pick narrows a release's archives to what to install from repo for the game: the asset shaped like the one the
// user chose before, else the ones Installable keeps, else among those the ones whose file list one of the game's
// loaders recognises as its mods. One asset means the choice is made; several are what the user picks from.
func (s *Service) pick(ctx context.Context, gameID, repo string, assets []github.Asset) []github.Asset {
	if len(assets) < 2 {
		return assets
	}
	s.choiceMu.Lock()
	shape := s.choices()[strings.ToLower(repo)]
	s.choiceMu.Unlock()
	if shape != "" {
		if i := slices.IndexFunc(assets, func(a github.Asset) bool { return github.Shape(a.Name) == shape }); i >= 0 {
			return assets[i : i+1]
		}
	}
	assets = github.Installable(assets, runtime.GOOS)
	if len(assets) < 2 {
		return assets
	}
	var shapes []loader.ModArchive
	if g, ok := gameInfo(gameID); ok {
		for _, l := range loader.For(g) {
			if m, ok := l.(loader.ModArchive); ok {
				shapes = append(shapes, m)
			}
		}
	}
	if len(shapes) == 0 {
		return assets
	}
	fit := slices.DeleteFunc(slices.Clone(assets), func(a github.Asset) bool {
		names, err := github.ZipNames(ctx, s.d.HTTP, a)
		return err != nil || !slices.ContainsFunc(shapes, func(m loader.ModArchive) bool { return m.ModArchive(names) })
	})
	if len(fit) == 0 {
		return assets
	}
	return fit
}

// GitHubAsset is what queueing repo's release at version (the newest when empty) for the game would download: the
// release and the asset, or the assets the user must choose from when the pick is ambiguous.
//
//wails:ignore
func (s *Service) GitHubAsset(ctx context.Context, gameID, repo, version string) (github.Release, []github.Asset, error) {
	rel, assets, err := s.release(ctx, Item{Repo: repo, Version: version})
	if err != nil {
		return rel, nil, err
	}
	return rel, s.pick(ctx, gameID, repo, assets), nil
}

// useGitHubFallback turns a Nexus item that is about to wait for a click into a download of the same version from
// the mod's GitHub releases, when it names a repo and that release has exactly one archive. It reports whether the
// item was switched; it asks at most once per item, so a missing release costs one lookup.
func (s *Service) useGitHubFallback(ctx context.Context, it Item) bool {
	if it.FallbackRepo == "" || it.Version == "" || it.Repo != "" {
		return false
	}
	s.mu.Lock()
	cur := s.find(it.ID)
	if cur == nil || cur.fallbackTried {
		s.mu.Unlock()
		return false
	}
	cur.fallbackTried = true
	s.mu.Unlock()
	rel, assets, err := s.GitHubAsset(ctx, it.Game, it.FallbackRepo, it.Version)
	if err != nil || len(assets) != 1 {
		return false
	}
	s.mu.Lock()
	cur = s.find(it.ID)
	if cur == nil || cur.State != StateWaitingClick {
		s.mu.Unlock()
		return false
	}
	cur.nexusFileName = cur.FileName
	cur.Repo, cur.Tag, cur.Asset, cur.FileName = it.FallbackRepo, rel.Tag, assets[0].Name, assets[0].Name
	cur.State, cur.key, cur.expires = StateQueued, "", 0
	s.mu.Unlock()
	s.publish(true)
	return true
}

// backToNexus undoes a GitHub fallback whose release turned out not to hold the mod: the item downloads from Nexus
// as it would have, and GitHub is not asked again.
func (s *Service) backToNexus(id string) {
	s.mu.Lock()
	cur := s.find(id)
	if cur == nil {
		s.mu.Unlock()
		return
	}
	log.Printf("queue: %s: the %s release does not contain %s; using Nexus", cur.Name, cur.Repo, cur.FallbackID)
	cur.Repo, cur.Tag, cur.Asset, cur.FileName = "", "", "", cur.nexusFileName
	cur.State, cur.Progress, cur.fallbackTried = StateQueued, 0, true
	s.mu.Unlock()
	s.publish(true)
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
	if it.Asset == "" {
		assets = s.pick(ctx, it.Game, it.Repo, assets)
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
	if err := verifyDigest(path, assets[i].Digest); err != nil {
		dropDownload(path)
		return fmt.Errorf("%s: %w", it.Asset, err)
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
	key, ids, err := s.d.Stage(ctx, it.Game, source, path)
	if err != nil {
		return err
	}
	if s.d.KeepArchives == nil || !s.d.KeepArchives() {
		dropDownload(path)
	}
	if it.FallbackID != "" && it.ModID != 0 && !slices.ContainsFunc(ids, func(id mod.ID) bool { return mod.Equal(id, it.FallbackID) }) {
		s.backToNexus(it.ID)
		return nil
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
	if it.chosenOverlay && it.Remap != nil {
		s.installMu.Lock()
		res, err := s.d.InstallStaged(it.Game, it.Profile, it.staged, it.Remap.Source.WithOverlay(it.chosenRoot, it.chosenTo))
		s.installMu.Unlock()
		return s.afterInstall(it.ID, res, err, false)
	}
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
