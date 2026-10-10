package sharesvc

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/source/patreon"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// Where a previewed mod stands. Installed means the target profile already has that file; Download and Dependency
// will be downloaded (a dependency being one the shared mods need and the link did not carry); Later is a file the
// mod dataset lacks, so its mod id and needs are known only after download; Unavailable cannot be installed.
const (
	StateInstalled   = "installed"
	StateDownload    = "download"
	StateDependency  = "dependency"
	StateLater       = "later"
	StateUnavailable = "unavailable"
)

// Why a mod is unavailable.
const (
	ReasonRemoved = "removed"
	ReasonNoFile  = "no-file"
	// ReasonPatreon is a mod from a Patreon post: open the post and save the file there.
	ReasonPatreon = "patreon"
)

// Kinds of Problem.
const (
	ProblemRemoved     = "removed"
	ProblemNoFile      = "no-file"
	ProblemBroken      = "broken"
	ProblemMissing     = "missing"
	ProblemFree        = "free"
	ProblemUnconfirmed = "unconfirmed"
)

// Mod is one file the import would bring. Key names it for Import's exclusions. A GitHub mod is Unverified: its
// trust check can only run after the download. Different marks a Nexus file replaced by another one of the mod.
type Mod struct {
	Key     string `json:"key"`
	Site    string `json:"site"`
	Name    string `json:"name"`
	Author  string `json:"author"`
	Version string `json:"version"`
	State   string `json:"state"`
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
	ModID   int    `json:"modId"`
	FileID  int    `json:"fileId"`
	Repo    string `json:"repo"`
	Tag     string `json:"tag"`
	Asset   string `json:"asset"`
	// fileName is the Nexus archive's name, which its entry records as the source name.
	fileName string
	// Package is a Thunderstore "Namespace-Name"; Version then names the version, or the newest when empty.
	Package    string                         `json:"package,omitempty"`
	PageURL    string                         `json:"pageUrl"`
	SizeKB     int64                          `json:"sizeKb"`
	Different  bool                           `json:"different"`
	Unverified bool                           `json:"unverified"`
	IDs        []mod.ID                       `json:"ids"`
	Disabled   []mod.ID                       `json:"disabled,omitempty"`
	Fomod      map[string]map[string][]string `json:"fomod,omitempty"`
	// Overlay is set on an optional file: where it goes in its main file, and whether it starts off.
	Overlay *share.Overlay `json:"overlay,omitempty"`
}

// Problem is something found before any download. Key is the mod it is about, when one is; Detail is a version
// or a mod id depending on Kind.
type Problem struct {
	Kind   string `json:"kind"`
	Key    string `json:"key"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
	URL    string `json:"url"`
}

// Preview is what an import would do. Settings counts the config files a .mortar file or an external profile carries.
type Preview struct {
	// Session names this preview; Import takes it back so it acts on exactly what the dialog shows.
	Session  string `json:"session"`
	Name     string `json:"name"`
	Notes    string `json:"notes"`
	Settings int    `json:"settings"`
	// Choices counts the dismissed problems and conflict winners the share carries.
	Choices int `json:"choices"`
	// SkippedSettings lists the external import's config files left out as too large, as mod id/path.
	SkippedSettings []string    `json:"skippedSettings,omitempty"`
	Mods            []Mod       `json:"mods"`
	Problems        []Problem   `json:"problems"`
	SignedIn        bool        `json:"signedIn"`
	Premium         bool        `json:"premium"`
	Replace         ReplacePlan `json:"replace"`
	// Collection is set for a Nexus collection import.
	Collection *CollectionInfo `json:"collection,omitempty"`
}

// Where a collection's installer choices and config files come from.
const (
	// DetailsArchive means Import downloads the curator's collection archive (Premium account) and applies its
	// FOMOD choices and bundled config files.
	DetailsArchive = "archive"
	// DetailsListed means only what Nexus lists without a download: the mod files, instructions and external resources.
	DetailsListed = "listed"
)

// CollectionInfo is what a collection import shows besides its mods. Instructions is the curator's text (markdown
// links and emphasis). External resources are never fetched: each is marked InstallYourself.
type CollectionInfo struct {
	Instructions string             `json:"instructions"`
	External     []ExternalResource `json:"external"`
	Details      string             `json:"details"`
}

// ExternalResource is a collection's resource that is not on Nexus. Type is "direct", "browse" or "manual".
type ExternalResource struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	URL             string `json:"url"`
	Version         string `json:"version"`
	Author          string `json:"author"`
	Optional        bool   `json:"optional"`
	InstallYourself bool   `json:"installYourself"`
}

// Site names of a Mod.
const (
	SiteNexus  = "nexus"
	SiteGitHub = "github"
	// SiteThunderstore is a Thunderstore package.
	SiteThunderstore = "thunderstore"
	SiteLocal        = "local"
	// SitePatreon is a Patreon post.
	SitePatreon = "patreon"
)

const (
	githubBase    = "https://github.com/"
	fetchParallel = 6
	categoryMain  = "MAIN"
	categoryOld   = "ARCHIVED"
)

// resolver looks one import up. Its answers come from the mod dataset and, when signed in, Nexus's file lists;
// a lookup that fails leaves the mod as the link named it rather than failing the preview.
type resolver struct {
	meta     problems.Meta
	files    func(ctx context.Context, t nexus.Title, modID int) ([]nexus.File, error)
	signedIn bool
	premium  bool
	env      problems.Environment
	// requirements reads what Nexus pages list as required; nil skips that check.
	requirements problems.RequirementsOf
	game         string
	stored       func(game, key string) bool
	storedKeys   map[string]bool
	// target is the profile the mods would join: what it has installed, and its entries by source.
	target    []profile.Entry
	installed []profile.Installed

	infos map[int]*modInfo
}

// modInfo is what is known about one Nexus mod page. checked means Nexus answered with the mod's files; failed
// means it could not be asked.
type modInfo struct {
	page    *meta.Page
	files   []nexus.File
	checked bool
	gone    bool
	failed  bool
}

// load looks up the mod pages not looked up yet.
func (r *resolver) load(ctx context.Context, modIDs []int) {
	if r.infos == nil {
		r.infos = map[int]*modInfo{}
	}
	var fresh []int
	for _, id := range modIDs {
		if _, ok := r.infos[id]; !ok {
			r.infos[id] = &modInfo{}
			fresh = append(fresh, id)
		}
	}
	var wg sync.WaitGroup
	slots := make(chan struct{}, fetchParallel)
	for _, id := range fresh {
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer func() { <-slots; wg.Done() }()
			info := r.infos[id]
			if page, err := r.meta.Page(ctx, id); err == nil {
				info.page = &page
			}
			if !r.signedIn {
				return
			}
			files, err := r.files(ctx, r.env.Nexus, id)
			var status *nexus.StatusError
			switch {
			case err == nil:
				info.files, info.checked = files, true
				info.gone = len(files) == 0
			case errors.As(err, &status) && status.Code == 404:
				info.checked, info.gone = true, true
			default:
				info.failed = true
			}
		}()
	}
	wg.Wait()
}

func (r *resolver) hasNexus(modID, fileID int) bool {
	return slices.ContainsFunc(r.target, func(e profile.Entry) bool {
		return e.Source.Kind == profile.KindNexus && e.Source.ModID == modID && e.Source.FileID == fileID
	})
}

func (r *resolver) hasGitHub(repo, tag, asset string) bool {
	return slices.ContainsFunc(r.target, func(e profile.Entry) bool {
		return e.Source.Kind == profile.KindGitHub && strings.EqualFold(e.Source.Repo, repo) && e.Source.Tag == tag && e.Source.Asset == asset
	})
}

func datasetFile(p *meta.Page, fileID int) *meta.File {
	if p == nil {
		return nil
	}
	for i := range p.Downloads {
		if p.Downloads[i].ID == int64(fileID) {
			return &p.Downloads[i]
		}
	}
	return nil
}

// substitute picks the file that stands in for one Nexus no longer offers: the MAIN file of the same version,
// else the mod's primary file.
func substitute(files []nexus.File, version string) *nexus.File {
	if version != "" {
		if i := slices.IndexFunc(files, func(f nexus.File) bool { return f.Category == categoryMain && f.Version == version }); i >= 0 {
			return &files[i]
		}
	}
	if i := slices.IndexFunc(files, func(f nexus.File) bool { return f.IsPrimary }); i >= 0 {
		return &files[i]
	}
	return nil
}

func nexusMod(domain string, modID, fileID int, state string) Mod {
	return Mod{Key: store.NexusKey(modID, fileID), Site: SiteNexus, ModID: modID, FileID: fileID, PageURL: nexus.ModURL(domain, modID), State: state, IDs: []mod.ID{}}
}

// nexus resolves one Nexus file.
func (r *resolver) nexus(modID, fileID int, state string) Mod {
	m := nexusMod(r.env.Nexus.Domain, modID, fileID, state)
	info := r.infos[modID]
	if info.page != nil {
		m.Name, m.Author = info.page.Name, info.page.Author
	}
	fallbackName := func(fileName string) {
		m.fileName = fileName
		m.Name = cmp.Or(m.Name, cleanFileName(fileName))
		m.Name = cmp.Or(m.Name, fmt.Sprintf("Nexus mod %d", modID))
	}
	if r.hasNexus(modID, fileID) {
		m.State = StateInstalled
		fallbackName("")
		return m
	}
	if r.storedKey(m.Key) {
		m.State = StateInstalled
		fallbackName("")
		return m
	}
	if info.gone {
		m.State, m.Reason = StateUnavailable, ReasonRemoved
		fallbackName("")
		return m
	}
	fileName := ""
	if info.checked {
		var listed *nexus.File
		if i := slices.IndexFunc(info.files, func(f nexus.File) bool { return f.FileID == fileID }); i >= 0 {
			listed = &info.files[i]
		}
		chosen := listed
		if listed == nil || listed.Category == categoryOld {
			version := ""
			if listed != nil {
				version = listed.Version
			} else if df := datasetFile(info.page, fileID); df != nil {
				version = df.Version
			}
			chosen = substitute(info.files, version)
			if chosen == nil {
				m.State, m.Reason = StateUnavailable, ReasonNoFile
				fallbackName("")
				return m
			}
			m.Different, m.FileID = true, chosen.FileID
			if r.hasNexus(modID, chosen.FileID) {
				m.State = StateInstalled
			}
		}
		m.Version, m.SizeKB, fileName = chosen.Version, chosen.SizeKB, chosen.FileName
	}
	df := datasetFile(info.page, m.FileID)
	if df != nil {
		m.Version = cmp.Or(m.Version, df.Version)
		m.SizeKB = cmp.Or(m.SizeKB, df.SizeInBytes/1024)
		for _, dm := range df.Mods {
			m.IDs = append(m.IDs, dm.ModID())
		}
	} else if m.State == StateDownload {
		m.State = StateLater
	}
	fallbackName(fileName)
	return m
}

func cleanFileName(name string) string {
	return strings.TrimSuffix(name, path.Ext(name))
}

func (r *resolver) github(ref share.Ref) Mod {
	repo, tag, asset := ref.GitHubParts()
	owner, name, _ := strings.Cut(repo, "/")
	m := Mod{
		Key: "github:" + ref.GitHub, Site: SiteGitHub, Name: name, Author: owner, Version: tag, Repo: repo, Tag: tag, Asset: asset,
		PageURL: githubBase + repo, State: StateDownload, Unverified: true, IDs: []mod.ID{},
		Disabled: append([]mod.ID{}, ref.Disabled...), Fomod: ref.Fomod,
	}
	if r.hasGitHub(repo, tag, asset) {
		m.State = StateInstalled
	} else if r.storedKey(github.Key(owner, name, tag, asset)) {
		m.State, r.storedKeys[m.Key] = StateInstalled, true
	}
	return m
}

func (r *resolver) hasPackage(pkg, version string) bool {
	return slices.ContainsFunc(r.target, func(e profile.Entry) bool {
		return e.Source.Kind == profile.KindThunderstore && strings.EqualFold(e.Source.Name, pkg) && (version == "" || e.Source.Version == version)
	})
}

func (r *resolver) thunderstore(ref share.Ref) Mod {
	ns, name, _ := strings.Cut(ref.Package, "-")
	m := Mod{
		Key: "thunderstore:" + ref.Package + "@" + ref.Version, Site: SiteThunderstore, Name: name, Author: ns, Version: ref.Version,
		Package: ref.Package, PageURL: thunderstore.Driver{}.ModPageURL(share.SourceKeys(r.game)["thunderstore"], ref.Package),
		State: StateDownload, IDs: []mod.ID{}, Disabled: append([]mod.ID{}, ref.Disabled...),
	}
	if r.hasPackage(ref.Package, ref.Version) {
		m.State = StateInstalled
	} else if ref.Version != "" && r.storedKey(store.PackageKey(ref.Package, ref.Version)) {
		m.State, r.storedKeys[m.Key] = StateInstalled, true
	}
	return m
}

// patreon resolves a file the sender saved from a Patreon post. Only the post travels, so it is installed when the
// profile already holds an entry from that post and otherwise unavailable until the receiver saves the file there.
func (r *resolver) patreon(ref share.Ref) Mod {
	m := Mod{
		Key: "patreon:" + ref.Patreon, Site: SitePatreon, Name: "Patreon post " + ref.Patreon, State: StateInstalled,
		PageURL: patreon.PostURL(ref.Patreon), IDs: []mod.ID{}, Disabled: append([]mod.ID{}, ref.Disabled...),
	}
	has := slices.ContainsFunc(r.target, func(e profile.Entry) bool {
		return e.Source.Kind == profile.KindPatreon && e.Source.Name == ref.Patreon
	})
	if !has {
		m.State, m.Reason = StateUnavailable, ReasonPatreon
	}
	return m
}

// local resolves an archive the sender installed from disk. Only a paired computer's copy of its store item can
// install it; Name is the archive's file name, which the entry keeps as its source.
func (r *resolver) local(ref share.Ref) Mod {
	m := Mod{
		Key: ref.Local, Site: SiteLocal, Name: ref.LocalName, State: StateInstalled, IDs: []mod.ID{},
		Disabled: append([]mod.ID{}, ref.Disabled...),
	}
	has := slices.ContainsFunc(r.target, func(e profile.Entry) bool { return e.StoreKey() == ref.Local })
	if !has && !r.storedKey(ref.Local) {
		m.State, m.Reason = StateUnavailable, ReasonNoFile
	}
	return m
}

func (r *resolver) storedKey(key string) bool {
	if r.stored == nil || !r.stored(r.game, key) {
		return false
	}
	r.storedKeys[key] = true
	return true
}

func nexusIDs(refs []share.Ref) []int {
	var ids []int
	for _, ref := range refs {
		if ref.GitHub == "" && ref.Package == "" && ref.Local == "" && ref.Patreon == "" && !slices.Contains(ids, ref.ModID) {
			ids = append(ids, ref.ModID)
		}
	}
	return ids
}

// resolve looks every shared file up, then the dependencies the shared mods need.
func (r *resolver) resolve(ctx context.Context, refs []share.Ref) ([]Mod, []Problem) {
	r.load(ctx, nexusIDs(refs))
	mods := make([]Mod, 0, len(refs))
	for _, ref := range refs {
		if ref.Local != "" {
			mods = append(mods, r.local(ref))
			continue
		}
		if ref.Patreon != "" {
			mods = append(mods, r.patreon(ref))
			continue
		}
		if ref.Package != "" {
			mods = append(mods, r.thunderstore(ref))
			continue
		}
		if ref.GitHub != "" {
			mods = append(mods, r.github(ref))
			continue
		}
		m := r.nexus(ref.ModID, ref.FileID, StateDownload)
		m.Disabled, m.Fomod, m.Overlay = append([]mod.ID{}, ref.Disabled...), ref.Fomod, ref.Overlay
		mods = append(mods, m)
	}
	deps, probs := r.dependencies(ctx, mods)
	mods = append(mods, deps...)
	return mods, r.findings(mods, probs)
}

// asInstalled presents a resolved mod to the problem checks as installed.
func asInstalled(m Mod, df *meta.File) []framework.Mod {
	var out []framework.Mod
	for _, dm := range df.Mods {
		mf := manifest.Manifest{Name: dm.Name, Version: dm.Version, UpdateKeys: dm.UpdateKeys}.WithModID(dm.ModID())
		for _, d := range dm.Dependencies {
			mf.Dependencies = append(mf.Dependencies, manifest.NewDependency(d.ModID(), d.MinimumVersion, d.Required))
		}
		out = append(out, framework.Mod{Key: m.Key, SourceKind: profile.KindNexus, Enabled: true, Manifest: mf})
	}
	return out
}

func (r *resolver) dependencies(ctx context.Context, mods []Mod) ([]Mod, []Problem) {
	var all []framework.Mod
	for _, i := range r.installed {
		all = append(all, framework.Mod{Key: i.Key, SourceKind: i.Source.Kind, Enabled: i.Enabled, Manifest: i.Manifest})
	}
	shared := map[string]Mod{}
	dependents := map[string]bool{}
	for _, m := range mods {
		if m.Site != SiteNexus || m.State != StateDownload {
			continue
		}
		if df := datasetFile(r.infos[m.ModID].page, m.FileID); df != nil {
			shared[m.Key] = m
			for _, i := range asInstalled(m, df) {
				dependents[i.ModID().Fold()] = true
				all = append(all, i)
			}
		}
	}
	found := problems.Check(ctx, r.meta, r.env, all, r.requirements)
	var deps []Mod
	var probs []Problem
	for _, miss := range found.Missing {
		if miss.External || !dependents[miss.DependentID.Fold()] || r.addDependency(ctx, miss.Where, mods, &deps) {
			continue
		}
		p := Problem{Kind: ProblemMissing, Name: miss.DependentName, Detail: miss.ID.Local()}
		if miss.Where != nil {
			p.URL = miss.Where.URL
		}
		probs = append(probs, p)
	}
	for _, b := range found.Broken {
		if _, ok := shared[b.Key]; ok {
			probs = append(probs, Problem{Kind: ProblemBroken, Key: b.Key, Name: b.Name, Detail: b.BrokeIn})
		}
	}
	return deps, probs
}

// addDependency adds the file that satisfies a missing dependency to deps, unless the import already has it, and
// reports whether the dependency is dealt with; false means nothing installable is known for it.
func (r *resolver) addDependency(ctx context.Context, where *problems.Ref, mods []Mod, deps *[]Mod) bool {
	if where == nil {
		return false
	}
	var m Mod
	switch {
	case where.Site == "Nexus" && where.FileID > 0:
		r.load(ctx, []int{where.PageID})
		m = r.nexus(where.PageID, int(where.FileID), StateDependency)
	case where.Site == "GitHub" && where.GitHub != "":
		owner, name, _ := strings.Cut(where.GitHub, "/")
		m = Mod{
			Key: "dep-github:" + where.GitHub, Site: SiteGitHub, Name: name, Author: owner, Repo: where.GitHub,
			PageURL: githubBase + where.GitHub, State: StateDependency, Unverified: true, IDs: []mod.ID{},
		}
	default:
		return false
	}
	if m.State == StateInstalled || slices.ContainsFunc(mods, func(x Mod) bool { return x.Key == m.Key }) || slices.ContainsFunc(*deps, func(x Mod) bool { return x.Key == m.Key }) {
		return true
	}
	*deps = append(*deps, m)
	return true
}

// findings adds what the resolved mods say about themselves to the problems already found.
func (r *resolver) findings(mods []Mod, probs []Problem) []Problem {
	out := []Problem{}
	unconfirmed, needsNexus := false, false
	for _, m := range mods {
		if m.Site == SiteNexus {
			unconfirmed = unconfirmed || r.infos[m.ModID].failed
			needsNexus = needsNexus || m.State == StateDownload || m.State == StateDependency || m.State == StateLater
		}
		if m.State != StateUnavailable {
			continue
		}
		kind := ProblemRemoved
		if m.Reason == ReasonNoFile {
			kind = ProblemNoFile
		}
		out = append(out, Problem{Kind: kind, Key: m.Key, Name: m.Name, URL: m.PageURL})
	}
	out = append(out, probs...)
	if r.signedIn && !r.premium && needsNexus {
		out = append(out, Problem{Kind: ProblemFree})
	}
	if unconfirmed {
		out = append(out, Problem{Kind: ProblemUnconfirmed})
	}
	return out
}
