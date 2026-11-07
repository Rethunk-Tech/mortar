package main

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/all"
)

// The version pickers' windows: internal/loader/smapi lastLoaderReleases and internal/loader/melonloader listedReleases.
const (
	smapiPicked = 10
	melonPicked = 40
)

var plainVersion = regexp.MustCompile(`^\d+(\.\d+)*$`)

func readPins(file string) ([]components.Pin, error) {
	body, err := safefs.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read pins: %w", err)
	}
	var f components.PinFile
	if err := json.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("decode pins: %w", err)
	}
	for _, p := range f.Pins {
		if err := p.Validate(); err != nil {
			return nil, err
		}
	}
	return f.Pins, nil
}

func writePins(file string, pins []components.Pin) error {
	slices.SortFunc(pins, func(a, b components.Pin) int {
		return cmp.Or(cmp.Compare(a.Kind, b.Kind), cmp.Compare(a.Game, b.Game), cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)), cmp.Compare(a.Version, b.Version))
	})
	body, err := json.MarshalIndent(components.PinFile{Pins: pins}, "", "  ")
	if err != nil {
		return err
	}
	return safefs.WriteFile(file, append(body, '\n'), 0o644)
}

type pinSet struct {
	file string
	pins []components.Pin
	have map[string]bool
}

func pinKey(p components.Pin) string {
	return p.Kind + "\x00" + p.Game + "\x00" + strings.ToLower(p.Name) + "\x00" + p.Version
}

func (s *pinSet) add(p components.Pin) error {
	if s.have[pinKey(p)] {
		return nil
	}
	s.have[pinKey(p)] = true
	s.pins = append(s.pins, p)
	return writePins(s.file, s.pins)
}

// refreshPins adds, for every release the loader pickers can offer, the SHA-256 of its real download to the pin file:
// SMAPI and MelonLoader from their GitHub releases, and each Thunderstore loader package from the versions its game's
// community index lists. Versions already pinned are not downloaded again, and each pin is saved as it is measured, so
// a run stopped by -budget resumes where it ended.
func refreshPins(ctx context.Context, o options) error {
	ctx, cancel := context.WithTimeout(ctx, o.budget)
	defer cancel()
	source, err := readSource(o.source)
	if err != nil {
		return err
	}
	existing, err := readPins(o.pins)
	if err != nil {
		return err
	}
	set := &pinSet{file: o.pins, pins: existing, have: map[string]bool{}}
	for _, p := range existing {
		set.have[pinKey(p)] = true
	}
	httpClient := &http.Client{Timeout: 5 * time.Minute}
	cacheDir, err := os.MkdirTemp("", "mortar-pins-cache-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(cacheDir) }()
	client := &github.Client{HTTP: httpClient, CacheDir: cacheDir}

	for _, item := range source.Components {
		if item.Kind != "loader" {
			continue
		}
		var window int
		var asset string
		switch item.Name {
		case "smapi":
			window = smapiPicked
		case "loader":
			window = melonPicked
		default:
			continue
		}
		asset = item.AssetPattern
		if err := pinGitHubLoader(ctx, client, set, item, window, asset); err != nil {
			return err
		}
	}
	if err := pinPacks(ctx, set, source.Games); err != nil {
		return err
	}
	log.Printf("pins: %d entries in %s", len(set.pins), o.pins)
	return nil
}

func pinGitHubLoader(ctx context.Context, client *github.Client, set *pinSet, item components.SourceComponent, window int, pattern string) error {
	releases, err := client.Releases(ctx, item.Source.Owner, item.Source.Repo)
	if err != nil {
		return fmt.Errorf("look up %s/%s: %w", item.Source.Owner, item.Source.Repo, err)
	}
	picked := 0
	for _, r := range releases {
		version := strings.TrimPrefix(r.Tag, "v")
		if r.Draft || r.Prerelease || !plainVersion.MatchString(version) {
			continue
		}
		name := strings.ReplaceAll(strings.ReplaceAll(pattern, "{version}", version), "{tag}", r.Tag)
		if !slices.ContainsFunc(r.Assets, func(a github.Asset) bool { return a.Name == name }) {
			continue
		}
		if picked++; picked > window {
			break
		}
		pin := components.Pin{Kind: components.PinComponent, Game: item.Game, Name: item.Name, Version: version}
		if set.have[pinKey(pin)] {
			continue
		}
		pinned := item
		pinned.Version = version
		component, err := resolve(ctx, client, pinned)
		if err != nil {
			return err
		}
		pin.SHA256 = component.SHA256
		if err := set.add(pin); err != nil {
			return err
		}
		log.Printf("pins: %s %s %s", item.Game, item.Name, version)
	}
	return nil
}

// pinPacks measures the packs of every Thunderstore loader. One game stands for each loader, package and major, since
// games sharing them are offered the same files.
func pinPacks(ctx context.Context, set *pinSet, games []components.GameInfo) error {
	scratch, err := os.MkdirTemp("", "mortar-pins-packs-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	for _, t := range packTargets(games) {
		versions, err := t.rel.Versions(ctx, t.game)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("pins: %s %s: %v", t.game.ID, t.game.LoaderPackage(), err)
			continue
		}
		stop, err := pinPackVersions(ctx, set, scratch, t, versions)
		if err != nil || stop {
			return err
		}
	}
	return nil
}

type packTarget struct {
	game components.GameInfo
	rel  loader.Releases
}

// packTargets lists each Thunderstore loader pack once per package and major, leaving out the loaders pinned as components.
func packTargets(games []components.GameInfo) []packTarget {
	var out []packTarget
	done := map[string]bool{}
	for _, g := range games {
		if _, ok := g.Source("thunderstore"); !ok {
			continue
		}
		for _, gl := range g.Loaders {
			l, ok := loader.Get(gl.ID)
			rel, releases := l.(loader.Releases)
			if !ok || !releases || gl.ID == "smapi" || gl.ID == "melonloader" {
				continue
			}
			key := gl.ID + "|" + g.LoaderPackage() + "|" + strconv.Itoa(g.LoaderMajor(gl.ID))
			if !done[key] {
				done[key] = true
				out = append(out, packTarget{game: g, rel: rel})
			}
		}
	}
	return out
}

// pinPackVersions downloads and hashes each version the set lacks; stop reports that the budget ran out.
func pinPackVersions(ctx context.Context, set *pinSet, scratch string, t packTarget, versions []string) (stop bool, err error) {
	for _, v := range versions {
		pin := components.Pin{Kind: components.PinPack, Name: t.game.LoaderPackage(), Version: v}
		if set.have[pinKey(pin)] {
			continue
		}
		dst := filepath.Join(scratch, "pack.zip")
		if err := t.rel.Fetch(ctx, t.game, v, dst); err != nil {
			if ctx.Err() != nil {
				log.Printf("pins: budget used; run again to continue")
				stop = true
				break
			}
			log.Printf("pins: %s %s %s: %v", t.game.ID, pin.Name, v, err)
			continue
		}
		sum, err := safefs.HashFile(dst, sha256.New())
		_ = safefs.Remove(dst)
		if err != nil {
			return false, err
		}
		pin.SHA256 = sum
		if err := set.add(pin); err != nil {
			return false, err
		}
		log.Printf("pins: %s %s", pin.Name, v)
	}
	return stop, nil
}
