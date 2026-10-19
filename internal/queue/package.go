package queue

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

// closureTimeout bounds the community index refresh a package request may wait on.
const closureTimeout = 2 * time.Minute

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
		exact := s.d.ExactDependencies != nil && s.d.ExactDependencies(r.Game, r.Profile)
		list, err := s.d.Closure(cctx, r.Game, roots, exact)
		cancel()
		if err != nil {
			return nil, err
		}
		inferred := inferPackageReasons(list, byID)
		for _, p := range list {
			id := p.Namespace + "-" + p.Name
			low := strings.ToLower(id)
			// The profile's loader provides the BepInEx pack, so it is never queued as a package.
			if components.IsLoaderPackage(id) {
				continue
			}
			dep, isRoot := byID[low]
			if isRoot && dep.InferReason {
				dep.Implicit = dep.Implicit || inferred[low]
			}
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
			dep.needs = packageNeeds(p.Dependencies)
			out = append(out, dep)
		}
	}
	return out, nil
}

// packageNeeds is the lowercase package ids of a package's "Namespace-Name-Version" dependencies.
func packageNeeds(deps []string) []string {
	var out []string
	for _, d := range deps {
		if i := strings.LastIndex(d, "-"); i > 0 {
			out = append(out, strings.ToLower(d[:i]))
		}
	}
	return out
}

// inferPackageReasons is which listed roots another listed root depends on, through the closure, by lowercase id.
func inferPackageReasons(list []thunderstore.Resolved, roots map[string]Request) map[string]bool {
	requires := map[string][]string{}
	for _, p := range list {
		id := strings.ToLower(p.Namespace + "-" + p.Name)
		for _, d := range p.Dependencies {
			if i := strings.LastIndex(d, "-"); i > 0 {
				requires[id] = append(requires[id], strings.ToLower(d[:i]))
			}
		}
	}
	return profile.InferImplicit(reachable(requires, roots))
}

// reachable maps each root to the other roots its closure reaches, so a root a dependency in between requires counts.
func reachable(requires map[string][]string, roots map[string]Request) map[string][]string {
	out := map[string][]string{}
	for root := range roots {
		seen := map[string]bool{}
		stack := slices.Clone(requires[root])
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[n] {
				continue
			}
			seen[n] = true
			if _, ok := roots[n]; ok && n != root {
				out[root] = append(out[root], n)
			}
			stack = append(stack, requires[n]...)
		}
	}
	return out
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
		Kind: cmp.Or(it.Source, profile.KindThunderstore), Name: it.Package, Version: it.Version, FileID: it.PackageFile, Digest: it.Digest,
		Picture: it.Picture, Category: it.Category,
	}
	if it.Source != "" && it.Name != it.Package {
		src.ModName = it.Name
	}
	if len(it.Disabled) > 0 {
		src = src.WithDisabled(it.Disabled)
	}
	return withIntent(it, src)
}

// cdnStallDefault is how long a Thunderstore download may receive nothing before the next CDN is tried.
const cdnStallDefault = 20 * time.Second

var errStalled = errors.New("the download stalled")

// cdnFailure reports an error a different CDN may not share: a connection that failed or a download that stalled.
func cdnFailure(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return false
	}
	var netErr net.Error
	return errors.Is(err, errStalled) || errors.As(err, &netErr)
}

// fetchPackage downloads a Thunderstore package from the first of the CDNs that serves it, moving on after a
// connection failure or a stall, and remembers the one that worked.
func (s *Service) fetchPackage(ctx context.Context, it Item, path string) error {
	hosts := s.d.CDNs
	if hosts == nil {
		hosts = thunderstore.CDNs
	}
	pinned := ""
	if s.d.ThunderstoreHost != nil {
		pinned = s.d.ThunderstoreHost()
	}
	s.mu.Lock()
	good := s.cdnGood
	s.mu.Unlock()
	order := thunderstore.CDNOrder(it.URL, hosts, pinned, good)
	if len(order) == 0 {
		return s.fetch(ctx, it, it.URL, path)
	}
	var err error
	for _, host := range order {
		u, _ := thunderstore.OnHost(it.URL, host)
		if err = s.fetchStalling(ctx, it, u, path); err == nil {
			s.mu.Lock()
			s.cdnGood = host
			s.mu.Unlock()
			return nil
		}
		if !cdnFailure(ctx, err) {
			return err
		}
		log.Printf("queue: %s: %v; trying the next Thunderstore CDN", it.Package, err)
	}
	return err
}

// fetchStalling is fetch that gives up with errStalled when no bytes arrive for the stall time.
func (s *Service) fetchStalling(ctx context.Context, it Item, url, path string) error {
	stall := cmp.Or(s.cdnStall, cdnStallDefault)
	cctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	timer := time.AfterFunc(stall, func() { cancel(errStalled) })
	defer timer.Stop()
	err := s.fetchTicking(cctx, it, url, path, func() { timer.Reset(stall) })
	if err != nil && errors.Is(context.Cause(cctx), errStalled) {
		return errStalled
	}
	return err
}

// downloadPackage fetches a package archive and installs it into the item's profile.
func (s *Service) downloadPackage(ctx context.Context, it Item) error {
	if s.d.InstallPackage == nil {
		return errors.New("cannot install Thunderstore packages here")
	}
	if it.URL == "" {
		return errors.New("the package has no download address: add it again")
	}
	if parked, err := s.guardSize(it, it.SizeKB<<10); parked || err != nil {
		return err
	}
	if err := os.MkdirAll(s.downloadRoot(), 0o700); err != nil {
		return err
	}
	path := s.dest(it.ID, it.FileName)
	if err := s.fetchPackage(ctx, it, path); err != nil {
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
	res, err := s.d.InstallPackage(ctx, it.Game, it.Profile, path, packageSource(it))
	s.installMu.Unlock()
	var dup *profile.DuplicateError
	if err == nil || errors.As(err, &dup) {
		s.dropDownloadUnlessKept(path)
	}
	if err == nil && it.Source == profile.KindVSModDB {
		s.queueRequired(ctx, it, res.Profile)
	}
	return s.afterInstall(it.ID, res, err, false)
}

// gameModules are the mod ids a Vintage Story install provides itself.
var gameModules = []string{"game", "survival", "creative"}

// queueRequired queues, from the same site, each mod the profile's mods require by id that none of them is: a site
// that publishes no dependencies leaves them to the files' own lists.
func (s *Service) queueRequired(ctx context.Context, it Item, p profile.Profile) {
	var reqs []Request
	for _, id := range missingRequired(p) {
		reqs = append(reqs, Request{Kind: KindDependency, Game: it.Game, Profile: it.Profile, Source: it.Source, Package: id})
	}
	if len(reqs) == 0 {
		return
	}
	// Add takes the queue's lock, which the worker calling this may be inside of; the item's own context ends with it.
	go func() {
		if _, err := s.Add(context.WithoutCancel(ctx), reqs); err != nil {
			log.Printf("queue: required mods of %s: %v", it.Package, err)
		}
	}()
}

// missingRequired is each mod id the profile's mods require that none of them is, once, in the order first required.
func missingRequired(p profile.Profile) []string {
	have := map[string]bool{}
	var want []string
	for _, e := range p.Entries {
		for _, c := range e.Mods {
			have[c.ModID] = true
			want = append(want, c.Requires...)
		}
	}
	var out []string
	for _, id := range want {
		if !have[id] && !slices.Contains(gameModules, id) {
			have[id] = true
			out = append(out, id)
		}
	}
	return out
}
