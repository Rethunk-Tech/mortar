// Package sharesvc is the window's side of sharing: it builds the links and .mortar files of a profile, turns a
// link or file into a preview that resolves every mod before anything downloads, and on the user's word creates
// the profile and fills the download queue.
package sharesvc

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/ids"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/migrate"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ArrivedEvent carries an Arrival to the window.
const ArrivedEvent = "share:arrived"

// DiscordLimit is the length of a message Discord accepts, which the link's meter is drawn against.
const DiscordLimit = 2000

// Kinds of Arrival.
const (
	ArrivalLink = "link"
	ArrivalFile = "file"
	ArrivalMod  = "mod"
)

const filePerm = 0o644

func historyBatchID() string {
	return "share-" + ids.New()
}

var (
	// ErrNoPreview means Import was asked for without a preview to act on.
	ErrNoPreview = errors.New("there is nothing to import: open a link or file first")
	// ErrStalePreview means Import named a preview other than the one the service holds.
	ErrStalePreview = errors.New("the preview changed: check it again before importing")
	// ErrSignedOut means the import has Nexus downloads and no account is signed in.
	ErrSignedOut = errors.New("sign in to Nexus Mods before importing mods from Nexus")
)

// Arrival is a link, a .mortar file, or a mod route that reached the app from outside. It is only ever shown in the
// import dialog or used to select a mod; nothing is imported by arriving.
type Arrival struct {
	ID    int    `json:"id"`
	Kind  string `json:"kind"`
	Value string `json:"value"`
	Game  string `json:"game,omitempty"`
	ModID int    `json:"modId,omitempty"`
}

// Queue is the download queue's part in an import.
type Queue interface {
	Add(ctx context.Context, reqs []queue.Request) ([]queue.Item, error)
}

// Deps is what the service needs from the rest of the app.
type Deps struct {
	Profiles *profile.Store
	Meta     problems.Meta
	// Files lists a Nexus mod's files; it is only called while SignedIn.
	Files    func(ctx context.Context, t nexus.Title, modID int) ([]nexus.File, error)
	SignedIn func() bool
	Premium  func() bool
	// CollectionArchive downloads a collection's curator archive by the API path Nexus reports; nil disables it.
	CollectionArchive func(ctx context.Context, downloadLink string) ([]byte, error)
	Env               func(game string) problems.Environment
	// NexusPages gives many Nexus pages' data in a few batched requests, for the requirements they list; nil skips them.
	NexusPages problems.NexusPagesOf
	Queue      Queue
	// Stored reports whether a source key is already available in Mortar's store.
	Stored func(game, key string) bool
	// Dir is the data folder holding pending-configs.json; empty keeps pending imports in memory only.
	Dir string
	// Emit is nil in tests that do not watch events.
	Emit func(name string, data any)
}

// Service is the sharing service.
type Service struct {
	d Deps
	// App is set after application.New, for dialogs and the clipboard.
	App *application.App
	// QueueChanged writes a pending import's config files as its mods finish installing, and forgets the import
	// once its downloads are settled. It is a field so the window cannot call it; wire it to queue.Deps.Changed.
	QueueChanged func(queue.State)

	// applyMu serialises queueChanged, which edits the pending imports and saves them.
	applyMu sync.Mutex
	mu      sync.Mutex
	nextID  int
	inbox   []Arrival
	current *session
	// gen counts previews started and discards, so a preview that finishes after a newer one does not replace it.
	gen     int
	pending []*pending
	retry   *time.Timer
	// lastExport is the collection draft ExportCollection last wrote.
	lastExport string
	lastQ      queue.State
	recheck    time.Duration
}

// NewService returns the service.
func NewService(d Deps) *Service {
	s := &Service{d: d, recheck: 5 * time.Second}
	s.pending = loadPending(d.Dir)
	s.QueueChanged = s.queueChanged
	return s
}

// session is the preview the dialog is showing, kept so Import acts on exactly what was shown.
type session struct {
	id          string
	game        string
	preview     Preview
	notes       string
	description string
	configs     []share.Config
	// loaderConfigs are the loader's own config files, written into the profile as soon as it exists.
	loaderConfigs []share.LoaderConfig
	origin        string
	collection    *profile.CollectionRef
	// archiveLink is the collection's archive path; archiveTried marks that Import already fetched it.
	archiveLink  string
	archiveTried bool
	// target is the profile the preview was resolved against; refs are what it resolved.
	target   string
	refs     []share.Ref
	stored   map[string]bool
	external []migrate.ModPreview
	groups   []share.FileGroup
}

func (s *Service) emit(name string, data any) {
	if s.d.Emit != nil {
		s.d.Emit(name, data)
	}
}

// --- Share ---

// Omitted is an enabled or switched-off mod a link cannot carry. Reason is "local", "off" or "unknown".
type Omitted struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Group is the shared mods of one source: "nexus" or "github".
type Group struct {
	Source string   `json:"source"`
	Count  int      `json:"count"`
	Mods   []string `json:"mods"`
}

// Info is what the Share dialog shows. Web and App are empty when the profile is too large for a link.
type Info struct {
	Name     string    `json:"name"`
	Web      string    `json:"web"`
	App      string    `json:"app"`
	Length   int       `json:"length"`
	Limit    int       `json:"limit"`
	Count    int       `json:"count"`
	TooLarge bool      `json:"tooLarge"`
	Groups   []Group   `json:"groups"`
	LeftOut  []Omitted `json:"leftOut"`
}

func (s *Service) find(game, id string) (profile.Profile, error) {
	all, err := s.d.Profiles.List(game)
	if err != nil {
		return profile.Profile{}, err
	}
	i := slices.IndexFunc(all, func(p profile.Profile) bool { return p.Error == "" && p.ID == id })
	if i < 0 {
		return profile.Profile{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("profile %s not found", id))
	}
	return all[i], nil
}

func entryNames(e profile.Entry, withDisabled bool) []string {
	var names []string
	for _, m := range e.Mods {
		if (withDisabled || !slices.Contains(e.Disabled, m.ID)) && m.Name != "" && !slices.Contains(names, m.Name) {
			names = append(names, m.Name)
		}
	}
	if len(names) == 0 {
		names = append(names, displayName(e))
	}
	return names
}

func displayName(e profile.Entry) string {
	if e.Source.Name != "" {
		return e.Source.Name
	}
	return e.Key
}

func reasonOf(e profile.Entry) string {
	if e.Source.Kind == profile.KindLocal {
		return "local"
	}
	return "unknown"
}

func withEntryKeys(p profile.Profile, keys []string) profile.Profile {
	if len(keys) == 0 {
		return p
	}
	want := make(map[string]bool, len(keys))
	for _, k := range keys {
		want[k] = true
	}
	out := p
	entries := make([]profile.Entry, 0, len(keys))
	for _, e := range p.Entries {
		if want[e.Key] {
			entries = append(entries, e)
		}
	}
	out.Entries = entries
	return out
}

// Share builds the link of a profile, and what it holds and leaves out. keys, when set, keeps only those entries.
func (s *Service) Share(game, profileID string, keys []string, include share.Include) (Info, error) {
	p, err := s.find(game, profileID)
	if err != nil {
		return Info{}, err
	}
	p = withEntryKeys(p, keys)
	return describe(game, p, s.d.Profiles.ShareFacts(game, p), include)
}

func describe(game string, p profile.Profile, facts profile.ShareFacts, include ...share.Include) (Info, error) {
	inc := share.DefaultInclude()
	if len(include) > 0 {
		inc = include[0]
	}
	_, left, off := share.Collect(p, inc)
	info := Info{Name: p.Name, Limit: DiscordLimit, Groups: []Group{}, LeftOut: []Omitted{}}
	res, err := share.Encode(game, p, facts, inc)
	switch {
	case err == nil:
		info.Web, info.App, info.Length = res.Web, res.App, len(res.Web)
	case errors.Is(err, share.ErrTooLarge):
		info.TooLarge = true
	default:
		return Info{}, err
	}
	byKey := map[string]profile.Entry{}
	for _, e := range p.Entries {
		byKey[e.Key] = e
	}
	leftKeys := map[string]bool{}
	for _, l := range left {
		leftKeys[l.Key] = true
		e := byKey[l.Key]
		info.LeftOut = append(info.LeftOut, Omitted{Name: strings.Join(entryNames(e, false), ", "), Reason: reasonOf(e)})
	}
	for _, key := range off {
		info.LeftOut = append(info.LeftOut, Omitted{Name: strings.Join(entryNames(byKey[key], true), ", "), Reason: "off"})
	}
	groups := map[string]*Group{}
	for _, e := range p.Entries {
		if e.Source.Bundled() || leftKeys[e.Key] || slices.Contains(off, e.Key) {
			continue
		}
		g := groups[e.Source.Kind]
		if g == nil {
			info.Groups = append(info.Groups, Group{Source: e.Source.Kind})
			g = &info.Groups[len(info.Groups)-1]
			groups[e.Source.Kind] = g
		}
		g.Count++
		info.Count++
		for _, name := range entryNames(e, false) {
			if !slices.Contains(g.Mods, name) {
				g.Mods = append(g.Mods, name)
			}
		}
	}
	return info, nil
}

// Saved is the outcome of SaveFile. Path is empty when the dialog was cancelled; Skipped lists the config files
// left out for their size or name.
type Saved struct {
	Path    string   `json:"path"`
	Skipped []string `json:"skipped"`
}

var fileNameUnsafe = strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")

// SaveFile asks where to save the profile as a .mortar file, and writes it there.
func (s *Service) SaveFile(game, profileID string, keys []string, include share.Include) (Saved, error) {
	p, err := s.find(game, profileID)
	if err != nil {
		return Saved{}, err
	}
	p = withEntryKeys(p, keys)
	modsDir, err := s.d.Profiles.ModsDir(game, profileID)
	if err != nil {
		return Saved{}, err
	}
	d := s.App.Dialog.SaveFile().
		SetFilename(fileNameUnsafe.Replace(p.Name)+".mortar").
		AddFilter("Mortar profile (.mortar)", "*.mortar")
	d.AttachToWindow(s.App.Window.Current())
	dest, err := d.PromptForSingleSelection()
	if err != nil || dest == "" {
		return Saved{Skipped: []string{}}, err
	}
	if !strings.EqualFold(filepath.Ext(dest), ".mortar") {
		dest += ".mortar"
	}
	var buf bytes.Buffer
	skipped, err := share.Write(&buf, game, p, modsDir, include)
	if err != nil {
		return Saved{}, err
	}
	if err := datadir.WriteFile(dest, buf.Bytes(), filePerm); err != nil {
		return Saved{}, err
	}
	return Saved{Path: dest, Skipped: append([]string{}, skipped...)}, nil
}

// ExportBytes writes the profile's .mortar payload in memory.
//
//wails:ignore
func (s *Service) ExportBytes(game, profileID string, include share.Include) ([]byte, []string, error) {
	p, err := s.find(game, profileID)
	if err != nil {
		return nil, nil, err
	}
	modsDir, err := s.d.Profiles.ModsDir(game, profileID)
	if err != nil {
		return nil, nil, err
	}
	var buf bytes.Buffer
	skipped, err := share.Write(&buf, game, p, modsDir, include)
	if err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), skipped, nil
}

// --- Import ---

// PickFile asks for a .mortar file and returns its path, or "" when the dialog is cancelled.
func (s *Service) PickFile() (string, error) {
	d := s.App.Dialog.OpenFile().
		SetTitle("Open a .mortar file").
		AddFilter("Mortar profile (.mortar)", "*.mortar")
	d.AttachToWindow(s.App.Window.Current())
	return d.PromptForSingleSelection()
}

// ReadClipboard returns the clipboard's text. It is called only when the user presses Paste in the import dialog.
func (s *Service) ReadClipboard() string {
	text, _ := s.App.Clipboard.Text()
	return text
}

// IsCollectionURL reports whether text is the link of a Nexus collection page, which Mortar opens in the import
// dialog like a share link.
func IsCollectionURL(text string) bool {
	_, slug, _, ok := parseCollectionURL(text)
	return ok && slug != ""
}

// PreviewLink reads a share link, or a bare payload, and resolves what it names against the profile it would join
// (profileID may be empty).
func (s *Service) PreviewLink(ctx context.Context, game, text, profileID string) (Preview, error) {
	if domain, slug, revision, ok := parseCollectionURL(text); ok {
		return s.previewCollection(ctx, game, domain, slug, revision, profileID)
	}
	if domain, modID, ok := nexus.ParseModURL(text); ok {
		return s.previewModPage(ctx, game, domain, modID, profileID)
	}
	shared, err := share.Parse(text)
	if err != nil {
		return Preview{}, err
	}
	if shared.Game != "" {
		if _, err := gamepkg.Require(shared.Game); err != nil {
			return Preview{}, err
		}
		if shared.Game != game {
			return Preview{}, usererr.New(usererr.Invalid, "this link is for "+shared.Game+", not "+game)
		}
	}
	return s.preview(ctx, game, shared, "", nil, profileID, profile.OriginLink)
}

// PreviewFile reads a .mortar file and resolves what it names.
func (s *Service) PreviewFile(ctx context.Context, game, file, profileID string) (Preview, error) {
	pv, err := share.Read(file)
	if err != nil {
		return Preview{}, err
	}
	return s.previewShared(ctx, game, pv, profileID)
}

// previewBytes reads a .mortar file received in memory and resolves what it names.
func (s *Service) previewBytes(ctx context.Context, game string, data []byte, profileID string) (Preview, error) {
	pv, err := share.ReadBytes(data)
	if err != nil {
		return Preview{}, err
	}
	return s.previewShared(ctx, game, pv, profileID)
}

// previewShared resolves a read .mortar file and keeps its description and groups on the session for the import.
func (s *Service) previewShared(ctx context.Context, game string, pv share.Preview, profileID string) (Preview, error) {
	out, err := s.preview(ctx, game, pv.Shared, pv.Notes, pv.Configs, profileID, profile.OriginMortar)
	if err != nil {
		return Preview{}, err
	}
	out.Settings += len(pv.LoaderConfigs)
	s.mu.Lock()
	if s.current != nil && s.current.id == out.Session {
		s.current.description = pv.Description
		s.current.groups = pv.Groups
		s.current.loaderConfigs = pv.LoaderConfigs
		s.current.preview.Settings = out.Settings
	}
	s.mu.Unlock()
	return out, nil
}

// PreviewData reads a base64-encoded .mortar payload received from another Mortar.
func (s *Service) PreviewData(ctx context.Context, game, encoded, profileID string) (Preview, error) {
	data, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return Preview{}, fmt.Errorf("%w: invalid payload", share.ErrBadFile)
	}
	return s.previewBytes(ctx, game, data, profileID)
}

// ImportData imports a .mortar payload received from another Mortar as the import dialog does by default: every
// mod it names, into a new profile named as the share.
//
//wails:ignore
func (s *Service) ImportData(ctx context.Context, game, encoded string) (Result, error) {
	pv, err := s.PreviewData(ctx, game, encoded, "")
	if err != nil {
		return Result{}, err
	}
	return s.Import(ctx, game, pv.Session, "", nil)
}

// ImportExternal imports another manager's profile as the import wizard does by default: every mod, into a new
// profile named as the external one.
//
//wails:ignore
func (s *Service) ImportExternal(ctx context.Context, game string, external migrate.ProfilePreview) (Result, error) {
	pv, err := s.PreviewExternal(ctx, game, external, "")
	if err != nil {
		return Result{}, err
	}
	return s.Import(ctx, game, pv.Session, "", nil)
}

// PreviewExternal resolves missing external mods through the normal import resolver and keeps staged folders local.
func (s *Service) PreviewExternal(ctx context.Context, game string, external migrate.ProfilePreview, profileID string) (Preview, error) {
	var refs []share.Ref
	for _, im := range external.Mods {
		if im.SourcePath == "" && im.NexusModID > 0 {
			refs = append(refs, share.Ref{ModID: im.NexusModID})
		}
	}
	configs, skipped := externalConfigs(external.Mods)
	out, err := s.preview(ctx, game, share.Shared{Name: external.Name, Entries: refs}, "", configs, profileID, "")
	if err != nil {
		return Preview{}, err
	}
	local := externalLocalMods(external.Mods)
	out.Mods = append(local, out.Mods...)
	out.SkippedSettings = skipped
	s.mu.Lock()
	if s.current != nil && s.current.id == out.Session {
		s.current.preview.Mods = append(local, s.current.preview.Mods...)
		s.current.preview.SkippedSettings = skipped
		s.current.external = slices.Clone(external.Mods)
	}
	s.mu.Unlock()
	return out, nil
}

// externalConfigs keeps each external mod's own config.json under the caps a .mortar file's configs have, and
// lists the ones left out.
func externalConfigs(mods []migrate.ModPreview) (configs []share.Config, skipped []string) {
	var total int64
	for _, im := range mods {
		if len(im.Config) == 0 || im.ID == "" {
			continue
		}
		total += int64(len(im.Config))
		if len(im.Config) > share.MaxConfigBytes || total > share.MaxConfigTotal || len(configs) >= share.MaxConfigFiles {
			skipped = append(skipped, im.ID.Local()+"/config.json")
			continue
		}
		configs = append(configs, share.Config{ID: im.ID, Path: "config.json", Data: im.Config})
	}
	return configs, skipped
}

func externalLocalMods(mods []migrate.ModPreview) []Mod {
	out := make([]Mod, 0, len(mods))
	for i, im := range mods {
		if im.SourcePath != "" {
			name := im.Name
			if name == "" {
				name = im.ID.Local()
			}
			out = append(out, Mod{
				Key: externalKey(i), Site: SiteLocal, Name: name, Version: im.Version,
				State: StateDownload, Enabled: im.Enabled, IDs: []mod.ID{im.ID},
			})
			continue
		}
		if im.NexusModID == 0 {
			name := im.Name
			if name == "" {
				name = im.ID.Local()
			}
			out = append(out, Mod{
				Key: externalKey(i), Site: SiteLocal, Name: name, Version: im.Version,
				State: StateUnavailable, Enabled: im.Enabled, Reason: ReasonNoFile, IDs: []mod.ID{im.ID},
			})
		}
	}
	return out
}

func externalKey(index int) string {
	return fmt.Sprintf("external:%d", index)
}

func (s *Service) preview(ctx context.Context, game string, shared share.Shared, notes string, configs []share.Config, profileID, origin string) (Preview, error) {
	s.mu.Lock()
	s.gen++
	gen := s.gen
	s.mu.Unlock()
	r, err := s.resolverFor(game, profileID)
	if err != nil {
		return Preview{}, err
	}
	mods, probs := r.resolve(ctx, shared.Entries)
	pv := Preview{
		Session: ids.New(), Name: shared.Name, Notes: notes, Settings: len(configs), Mods: mods, Problems: probs,
		SignedIn: r.signedIn, Premium: r.premium,
	}
	if profileID != "" {
		if p, err := s.find(game, profileID); err == nil {
			pv.Replace = PlanReplace(p, mods)
		}
	}
	s.mu.Lock()
	if s.gen == gen {
		s.current = &session{
			id: pv.Session, game: game, preview: pv, notes: notes, configs: configs, origin: origin, target: profileID, refs: shared.Entries,
			stored: maps.Clone(r.storedKeys),
		}
	}
	s.mu.Unlock()
	return pv, nil
}

// resolverFor resolves against profileID, whose mods count as installed, or against nothing when it is empty.
func (s *Service) resolverFor(game, profileID string) (*resolver, error) {
	r := &resolver{
		meta: s.d.Meta, files: s.d.Files, signedIn: s.d.SignedIn(), premium: s.d.Premium(), env: s.d.Env(game),
		game: game, stored: s.d.Stored, storedKeys: map[string]bool{}, requirements: s.d.NexusPages.Requirements(game),
	}
	if profileID == "" {
		return r, nil
	}
	p, err := s.find(game, profileID)
	if err != nil {
		return nil, err
	}
	r.target = p.Entries
	if r.installed, err = s.d.Profiles.Installed(game, profileID); err != nil {
		return nil, err
	}
	return r, nil
}

// writeLoaderConfigs writes the shared loader config files that fall under the profile's own loader's config folders.
// Unless overwrite, a file the profile already has is kept, as a mod's config is for a mod it already holds.
func (s *Service) writeLoaderConfigs(game, profileID string, configs []share.LoaderConfig, overwrite bool) error {
	if len(configs) == 0 {
		return nil
	}
	roots := share.LoaderConfigRoots(game, s.d.Profiles.LoaderID(game, profileID))
	dir, err := s.d.Profiles.ProfileDir(game, profileID)
	if err != nil {
		return err
	}
	files := map[string][]byte{}
	for _, c := range configs {
		if !share.UnderRoots(c.Path, roots) {
			continue
		}
		if !overwrite && exists(filepath.Join(dir, filepath.FromSlash(c.Path))) {
			continue
		}
		files[c.Path] = c.Data
	}
	if len(files) == 0 {
		return nil
	}
	return s.d.Profiles.WriteFiles(game, profileID, files)
}

// Discard forgets the preview; closing the import dialog leaves nothing behind.
func (s *Service) Discard() {
	s.mu.Lock()
	s.gen++
	s.current = nil
	s.mu.Unlock()
}

// Result is what Import did.
type Result struct {
	Profile profile.Profile `json:"profile"`
	Queued  int             `json:"queued"`
	BatchID string          `json:"batchId,omitempty"`
	// Items are the ids of the queue items the import waits on, dependencies included.
	Items []string `json:"items"`
	// Collection is set when a collection import used the curator's archive.
	Collection *CollectionApplied `json:"collection,omitempty"`
}

// storedEntry is a store item an import places directly, with the source its entry records.
type storedEntry struct {
	key    string
	source profile.Source
}

func requestFor(game, profileID string, m Mod) queue.Request {
	kind := queue.KindInstall
	if m.State == StateDependency {
		kind = queue.KindDependency
	}
	r := queue.Request{Kind: kind, Game: game, Profile: profileID, Name: m.Name, Version: m.Version}
	r.Disabled, r.Fomod = m.Disabled, m.Fomod
	if m.Site == SiteThunderstore {
		r.Package = m.Package
		return r
	}
	if m.Site == SiteGitHub {
		r.Repo, r.Tag, r.Asset = m.Repo, m.Tag, m.Asset
		return r
	}
	r.ModID, r.FileID = m.ModID, m.FileID
	if o := m.Overlay; o != nil {
		r.Overlay = &queue.OverlayPlace{From: o.From, To: o.To, Off: o.Off}
	}
	return r
}

// unavailableNote lists the mods an import could not bring, for the new profile's notes.
func unavailableNote(mods []Mod) string {
	var b strings.Builder
	for _, m := range mods {
		if m.State != StateUnavailable {
			continue
		}
		if b.Len() == 0 {
			b.WriteString("Not imported, no longer available:")
		}
		fmt.Fprintf(&b, "\n- %s (%s)", m.Name, m.PageURL)
	}
	return b.String()
}

func joinNotes(parts ...string) string {
	var kept []string
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "\n\n")
}

func (s *Service) applySharedEntryNotes(game, profileID string, refs []share.Ref) error {
	p, err := s.find(game, profileID)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		if ref.Note == "" && len(ref.Tags) == 0 {
			continue
		}
		var key string
		for _, e := range p.Entries {
			if ref.MatchesEntry(e) {
				key = e.Key
				break
			}
		}
		if key == "" {
			continue
		}
		var patch profile.Entry
		share.ImportEntryNotes(&patch, ref)
		p, err = s.d.Profiles.SetEntryNoteTags(game, profileID, key, patch.Note, patch.Tags)
		if err != nil {
			return err
		}
	}
	return nil
}

// Import queues everything the preview named session lists as available, except the excluded keys, into
// profileID, or into a new profile named as the share (with " (2)" and so on when that name is taken) when profileID
// is empty. Mods the preview marked unavailable are listed in the new profile's notes. A .mortar file's config files
// are written once their mods are installed, except for mods the profile already had, whose config is the user's.
// A preview resolved against another profile than profileID is resolved again, so a new profile made from a preview
// of the open one still gets the mods the open one has.
func (s *Service) Import(ctx context.Context, game, session, profileID string, exclude []string) (Result, error) {
	return s.importWithBatch(ctx, game, session, profileID, exclude, "", false)
}

// replace makes the share's order the profile's: its mods keep the order they were shared in, wherever they land.
func (s *Service) importWithBatch(ctx context.Context, game, session, profileID string, exclude []string, batchID string, replace bool) (res Result, err error) {
	s.mu.Lock()
	cur := s.current
	switch {
	case cur == nil || cur.game != game:
		s.mu.Unlock()
		return Result{}, ErrNoPreview
	case cur.id != session:
		s.mu.Unlock()
		return Result{}, ErrStalePreview
	}
	// Taking the preview makes a second submit of it fail instead of creating a second profile.
	s.current = nil
	s.mu.Unlock()
	defer func() {
		if err != nil {
			s.mu.Lock()
			if s.current == nil {
				s.current = cur
			}
			s.mu.Unlock()
		}
	}()
	applied := s.applyArchive(ctx, cur)
	defer func() { res.Collection = applied }()
	mods := cur.preview.Mods
	stored := cur.stored
	if profileID != cur.target {
		r, err := s.resolverFor(game, profileID)
		if err != nil {
			return Result{}, err
		}
		mods, _ = r.resolve(ctx, cur.refs)
		mods = append(externalLocalMods(cur.external), mods...)
		stored = r.storedKeys
	}
	var reqs []queue.Request
	var wanted []wantedFile
	var local []profile.ExternalMod
	// fromStore are packages and archives already in the store, a paired computer's copies: they are placed from
	// there, not downloaded again.
	var fromStore []storedEntry
	external := make(map[string]migrate.ModPreview, len(cur.external))
	for i, im := range cur.external {
		external[externalKey(i)] = im
	}
	for _, m := range mods {
		if slices.Contains(exclude, m.Key) || m.State == StateUnavailable ||
			(m.State == StateInstalled && !stored[m.Key]) {
			continue
		}
		if m.Site == SiteLocal {
			if im, ok := external[m.Key]; ok {
				local = append(local, profile.ExternalMod{SourcePath: im.SourcePath, ID: im.ID, Enabled: im.Enabled})
			} else if stored[m.Key] {
				fromStore = append(fromStore, storedEntry{key: m.Key, source: profile.Source{Kind: profile.KindLocal, Name: m.Name}.WithDisabled(m.Disabled)})
			}
			continue
		}
		if m.Site == SiteThunderstore && stored[m.Key] {
			src := profile.Source{Kind: profile.KindThunderstore, Name: m.Package, Version: m.Version}.WithDisabled(m.Disabled)
			fromStore = append(fromStore, storedEntry{key: store.PackageKey(m.Package, m.Version), source: src})
			continue
		}
		reqs = append(reqs, requestFor(game, "", m))
		wanted = append(wanted, wantedOf(m))
	}
	if slices.ContainsFunc(reqs, func(r queue.Request) bool { return r.Repo == "" && r.Package == "" }) && !s.d.SignedIn() {
		return Result{}, ErrSignedOut
	}
	configs := cur.configs
	created := profileID == ""
	if created {
		existing, err := s.d.Profiles.List(game)
		if err != nil {
			return Result{}, err
		}
		names := make([]string, 0, len(existing))
		for _, e := range existing {
			if e.Error == "" {
				names = append(names, e.Name)
			}
		}
		p, err := s.d.Profiles.Create(game, profile.UniqueName(names, cur.preview.Name))
		if err != nil {
			return Result{}, err
		}
		if cur.origin != "" {
			stamped, err := s.d.Profiles.SetOrigin(game, p.ID, cur.origin, "")
			if err != nil {
				return Result{}, errors.Join(err, s.d.Profiles.Delete(game, p.ID))
			}
			p = stamped
		}
		if cur.collection != nil {
			stamped, err := s.d.Profiles.SetCollection(game, p.ID, *cur.collection)
			if err != nil {
				return Result{}, errors.Join(err, s.d.Profiles.Delete(game, p.ID))
			}
			p = stamped
		}
		notes := joinNotes(cur.notes, unavailableNote(mods))
		if utf8.RuneCountInString(notes) > profile.MaxNotes {
			notes = string([]rune(notes)[:profile.MaxNotes])
		}
		if notes != "" {
			withNotes, err := s.d.Profiles.SetNotes(game, p.ID, notes)
			if err != nil {
				return Result{}, errors.Join(err, s.d.Profiles.Delete(game, p.ID))
			}
			p = withNotes
		}
		if cur.description != "" {
			withDesc, err := s.d.Profiles.SetAppearance(game, p.ID, "", "", cur.description)
			if err != nil {
				return Result{}, errors.Join(err, s.d.Profiles.Delete(game, p.ID))
			}
			p = withDesc
		}
		res.Profile = p
		profileID = p.ID
	} else {
		p, err := s.find(game, profileID)
		if err != nil {
			return Result{}, err
		}
		if cur.collection != nil {
			stamped, err := s.d.Profiles.SetCollection(game, profileID, *cur.collection)
			if err != nil {
				return Result{}, err
			}
			p = stamped
		}
		res.Profile = p
		configs = slices.DeleteFunc(slices.Clone(configs), func(c share.Config) bool {
			_, _, held := p.FindMod("", c.ID)
			return held
		})
	}
	if err := s.writeLoaderConfigs(game, profileID, cur.loaderConfigs, created || replace); err != nil {
		if created {
			err = errors.Join(err, s.d.Profiles.Delete(game, profileID))
		}
		return Result{}, err
	}
	if len(local) > 0 || len(reqs) > 0 || len(fromStore) > 0 {
		if batchID == "" {
			batchID = historyBatchID()
		}
		if err := s.d.Profiles.OpenHistoryBatch(game, profileID, batchID); err != nil {
			if created {
				err = errors.Join(err, s.d.Profiles.Delete(game, profileID))
			}
			return Result{}, err
		}
	}
	var change string
	for _, e := range fromStore {
		added, err := s.d.Profiles.AddEntry(game, profileID, e.key, e.source)
		if err != nil {
			_ = s.d.Profiles.CloseHistoryBatch(game, profileID)
			if created {
				err = errors.Join(err, s.d.Profiles.Delete(game, profileID))
			}
			return Result{}, err
		}
		change = added.LastChange
	}
	if len(local) > 0 {
		if err := s.d.Profiles.ImportExternalMods(game, profileID, local); err != nil {
			if created {
				err = errors.Join(err, s.d.Profiles.Delete(game, profileID))
			}
			return Result{}, err
		}
	}
	if len(local) > 0 || len(fromStore) > 0 {
		if p, err := s.find(game, profileID); err == nil {
			res.Profile = p
			res.Profile.LastChange = change
		}
		// Copied folders and store items are on the profile now, so their configs land at once; the rest wait for
		// their downloads.
		if len(configs) > 0 {
			var written []mod.ID
			err := s.d.Profiles.InMods(game, profileID, func(prof profile.Profile, modsDir string) error {
				var err error
				written, err = share.Apply(modsDir, prof.Entries, configs)
				return err
			})
			if err != nil {
				if created {
					err = errors.Join(err, s.d.Profiles.Delete(game, profileID))
				}
				return Result{}, err
			}
			configs = slices.DeleteFunc(slices.Clone(configs), func(c share.Config) bool {
				return slices.ContainsFunc(written, func(id mod.ID) bool { return mod.Equal(id, c.ID) })
			})
		}
	}
	for i := range reqs {
		reqs[i].Profile, reqs[i].BatchID = profileID, batchID
	}
	res.Items = []string{}
	if len(reqs) > 0 {
		items, err := s.d.Queue.Add(ctx, reqs)
		if err != nil {
			_ = s.d.Profiles.CloseHistoryBatch(game, profileID)
			if created {
				err = errors.Join(err, s.d.Profiles.Delete(game, profileID))
			}
			return Result{}, err
		}
		for _, it := range items {
			res.Items = append(res.Items, it.ID)
		}
	} else if batchID != "" {
		_ = s.d.Profiles.CloseHistoryBatch(game, profileID)
	}
	res.Queued = len(reqs)
	res.BatchID = batchID
	if err := s.applySharedEntryNotes(game, profileID, cur.refs); err != nil {
		if created {
			err = errors.Join(err, s.d.Profiles.Delete(game, profileID))
		}
		return Result{}, err
	}
	if err := s.applySharedGroups(game, profileID, cur.groups); err != nil {
		if created {
			err = errors.Join(err, s.d.Profiles.Delete(game, profileID))
		}
		return Result{}, err
	}
	var placed []string
	if replace {
		if err := s.d.Profiles.FollowOrder(game, profileID, refRank(cur.refs)); err != nil {
			return Result{}, err
		}
		if placed, err = s.d.Profiles.PlaceArrivals(game, profileID, refRank(cur.refs), nil); err != nil {
			return Result{}, err
		}
	}
	if len(reqs) > 0 {
		// savePending marshals every pending import, which queueChanged edits under applyMu.
		s.applyMu.Lock()
		s.mu.Lock()
		s.pending = append(s.pending, &pending{
			Game: game, Profile: profileID, BatchID: batchID, Wanted: wanted, Configs: configs,
			Refs: slices.Clone(cur.refs), Groups: slices.Clone(cur.groups), Follow: replace || created, Placed: placed,
		})
		s.mu.Unlock()
		s.savePending()
		s.applyMu.Unlock()
	}
	return res, nil
}

// Replace makes profileID match the preview: missing mods are queued as Import does, entries not in the share
// are removed, and local-only mods not in the share are kept. profileID must be the preview's target.
func (s *Service) Replace(ctx context.Context, game, session, profileID string, exclude []string) (Result, error) {
	if profileID == "" {
		return Result{}, errors.New("replace needs a profile")
	}
	s.mu.Lock()
	cur := s.current
	s.mu.Unlock()
	if cur == nil || cur.game != game || cur.id != session {
		if cur == nil || cur.game != game {
			return Result{}, ErrNoPreview
		}
		return Result{}, ErrStalePreview
	}
	mods := cur.preview.Mods
	p, err := s.find(game, profileID)
	if err != nil {
		return Result{}, err
	}
	plan := PlanReplace(p, mods)
	batchID := historyBatchID()
	if err := s.d.Profiles.OpenHistoryBatch(game, profileID, batchID); err != nil {
		return Result{}, err
	}
	if len(plan.RemoveKeys) > 0 {
		if _, err := s.d.Profiles.RemoveEntries(game, profileID, plan.RemoveKeys); err != nil {
			_ = s.d.Profiles.CloseHistoryBatch(game, profileID)
			return Result{}, err
		}
	}
	return s.importWithBatch(ctx, game, session, profileID, exclude, batchID, true)
}

// --- Config files ---

// wantedFile names a file an import queued, as the profile's entry for it will record its source.
type wantedFile struct {
	ModID  int    `json:"modId"`
	FileID int    `json:"fileId"`
	Repo   string `json:"repo"`
	Tag    string `json:"tag"`
	Asset  string `json:"asset"`
	// Package is a Thunderstore "Namespace-Name".
	Package string `json:"package,omitempty"`
}

func wantedOf(m Mod) wantedFile {
	return wantedFile{ModID: m.ModID, FileID: m.FileID, Repo: m.Repo, Tag: m.Tag, Asset: m.Asset, Package: m.Package}
}

func (w wantedFile) entry(e profile.Entry) bool {
	if w.Package != "" {
		return e.Source.Kind == profile.KindThunderstore && strings.EqualFold(e.Source.Name, w.Package)
	}
	if w.Repo != "" {
		return e.Source.Kind == profile.KindGitHub && strings.EqualFold(e.Source.Repo, w.Repo) && (w.Tag == "" || e.Source.Tag == w.Tag) &&
			(w.Asset == "" || e.Source.Asset == w.Asset)
	}
	return e.Source.Kind == profile.KindNexus && e.Source.ModID == w.ModID && e.Source.FileID == w.FileID
}

func (w wantedFile) item(it queue.Item) bool {
	if w.Package != "" {
		return strings.EqualFold(it.Package, w.Package)
	}
	if w.Repo != "" {
		return strings.EqualFold(it.Repo, w.Repo) && (w.Tag == "" || it.Tag == w.Tag) &&
			(w.Asset == "" || it.Asset == w.Asset)
	}
	return it.ModID == w.ModID && it.FileID == w.FileID
}

// pending is a .mortar file's config files waiting for the mods they belong to.
type pending struct {
	Game    string            `json:"game"`
	Profile string            `json:"profile"`
	BatchID string            `json:"batchId"`
	Wanted  []wantedFile      `json:"wanted"`
	Configs []share.Config    `json:"configs"`
	Refs    []share.Ref       `json:"refs,omitempty"`
	Groups  []share.FileGroup `json:"groups,omitempty"`
	// Follow puts each mod that lands at its place in Refs instead of the end.
	Follow bool `json:"follow,omitempty"`
	// Placed are the keys of entries already in their shared place; the player may have moved them since.
	Placed []string `json:"placed,omitempty"`
	// ordered is the set of finished downloads the order was last fixed for; it starts over with the process.
	ordered string
	// seen is the set of finished downloads the configs were last applied for; it starts over with the process.
	seen string
}

// refRank ranks an entry by the first of refs it matches.
func refRank(refs []share.Ref) func(profile.Entry) (int, bool) {
	return func(e profile.Entry) (int, bool) {
		i := slices.IndexFunc(refs, func(r share.Ref) bool { return r.MatchesEntry(e) })
		return i, i >= 0
	}
}

func (p *pending) wants(e profile.Entry) bool {
	return slices.ContainsFunc(p.Wanted, func(w wantedFile) bool { return w.entry(e) })
}

func (s *Service) wantedInstalled(p *pending) bool {
	if len(p.Wanted) == 0 {
		return false
	}
	all, err := s.d.Profiles.List(p.Game)
	if err != nil {
		return false
	}
	var entries []profile.Entry
	for _, pr := range all {
		if pr.Error == "" && pr.ID == p.Profile {
			entries = pr.Entries
			break
		}
	}
	for _, w := range p.Wanted {
		if !slices.ContainsFunc(entries, w.entry) {
			return false
		}
	}
	return true
}

// queueChanged runs on every full queue state, which progress ticks do not publish, and saves only when a pending import changed.
func (s *Service) queueChanged(st queue.State) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	s.mu.Lock()
	todo := slices.Clone(s.pending)
	s.mu.Unlock()
	changed := false
	for _, p := range todo {
		var done []string
		settled := 0
		for _, it := range st.Items {
			if it.Game != p.Game || it.Profile != p.Profile {
				continue
			}
			if p.BatchID != "" && it.BatchID != p.BatchID {
				continue
			}
			if slices.ContainsFunc(p.Wanted, func(w wantedFile) bool { return w.item(it) }) &&
				(it.State == queue.StateDone || it.State == queue.StateFailed || it.State == queue.StateSkipped || it.State == queue.StateCancelled) {
				settled++
			}
			if it.State == queue.StateDone && slices.ContainsFunc(p.Wanted, func(w wantedFile) bool { return w.item(it) }) {
				done = append(done, it.ID)
			}
		}
		// The queue drops old finished items, so apply also when every wanted file is already on the profile.
		slices.Sort(done)
		seen := strings.Join(done, ",")
		if s.wantedInstalled(p) {
			seen = "on-profile"
		}
		if len(p.Configs) > 0 && seen != p.seen && (len(done) > 0 || settled >= len(p.Wanted) || seen == "on-profile") {
			p.seen = seen
			before := len(p.Configs)
			s.apply(p)
			changed = changed || len(p.Configs) != before
		}
		if p.Follow && len(p.Refs) > 0 && seen != p.ordered && (len(done) > 0 || settled >= len(p.Wanted) || seen == "on-profile") {
			if placed, err := s.d.Profiles.PlaceArrivals(p.Game, p.Profile, refRank(p.Refs), p.Placed); err != nil {
				log.Printf("share: order %s/%s as shared: %v", p.Game, p.Profile, err)
			} else {
				changed = changed || len(placed) != len(p.Placed)
				p.Placed, p.ordered = placed, seen
			}
		}
		if len(p.Refs) > 0 && (seen == "on-profile" || settled >= len(p.Wanted)) {
			if err := s.applySharedEntryNotes(p.Game, p.Profile, p.Refs); err == nil {
				p.Refs = nil
				changed = true
			}
		}
		if len(p.Groups) > 0 && (seen == "on-profile" || settled >= len(p.Wanted)) {
			if err := s.applySharedGroups(p.Game, p.Profile, p.Groups); err == nil {
				p.Groups = nil
				changed = true
			}
		}
		// Configs stay until they land; queue eviction must not drop them.
		if len(p.Configs) == 0 && len(p.Refs) == 0 && len(p.Groups) == 0 {
			s.mu.Lock()
			s.pending = slices.DeleteFunc(s.pending, func(x *pending) bool { return x == p })
			s.mu.Unlock()
			changed = true
		}
	}
	if changed {
		s.savePending()
	}
	s.armPendingRetry(st)
}

func (s *Service) armPendingRetry(st queue.State) {
	s.mu.Lock()
	s.lastQ = st
	waiting := slices.ContainsFunc(s.pending, func(p *pending) bool { return len(p.Configs) > 0 })
	if !waiting {
		if s.retry != nil {
			s.retry.Stop()
			s.retry = nil
		}
		s.mu.Unlock()
		return
	}
	if s.retry != nil {
		s.mu.Unlock()
		return
	}
	s.retry = time.AfterFunc(s.recheck, s.retryPending)
	s.mu.Unlock()
}

func (s *Service) retryPending() {
	s.mu.Lock()
	s.retry = nil
	st := s.lastQ
	s.mu.Unlock()
	s.queueChanged(st)
}

// errRunning stops an apply that found the game running the profile, under the lock a launch takes.
var errRunning = usererr.New(usererr.Busy, "the game is running the profile")

// apply writes the config files whose mods are installed, and keeps the rest for later. While the game runs the
// profile it waits for the next change.
func (s *Service) apply(p *pending) {
	var written []mod.ID
	var applyErr error
	err := s.d.Profiles.InMods(p.Game, p.Profile, func(prof profile.Profile, modsDir string) error {
		if r := s.d.Profiles.Running; r != nil && r(p.Game, p.Profile) {
			return errRunning
		}
		entries := slices.DeleteFunc(slices.Clone(prof.Entries), func(e profile.Entry) bool { return !p.wants(e) })
		written, applyErr = share.Apply(modsDir, entries, p.Configs)
		return applyErr
	})
	if err != nil || applyErr != nil {
		p.seen = ""
	}
	if err != nil {
		return
	}
	p.Configs = slices.DeleteFunc(p.Configs, func(c share.Config) bool {
		return slices.ContainsFunc(written, func(id mod.ID) bool { return mod.Equal(id, c.ID) })
	})
}

// --- Links and files from outside ---

const (
	webLinkPrefix = "https://mortar.rethunk.tech/"
	appLinkPrefix = "mortar://"
	fileScheme    = "file://"
)

// fileURLPath turns a file:// URL into a filesystem path. Windows drive URLs are /C:/... after parse; a host is a UNC share.
func fileURLPath(arg string) string {
	rest, ok := strings.CutPrefix(arg, fileScheme)
	if !ok {
		return arg
	}
	u, err := url.Parse(fileScheme + rest)
	if err != nil {
		return arg
	}
	path, err := url.PathUnescape(u.Path)
	if err != nil {
		path = u.Path
	}
	if u.Host != "" {
		return filepath.FromSlash(`\\` + u.Host + path)
	}
	if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return filepath.FromSlash(path)
}

// classify says whether a launch argument is a mod route, share link, or .mortar file. It looks no further than the
// form: the import dialog parses and previews links, and refuses what is not a share.
func classify(arg string) (Arrival, bool) {
	if route, ok := parseModRoute(arg); ok {
		return Arrival{Kind: ArrivalMod, Value: arg, Game: route.game, ModID: route.modID}, true
	}
	if strings.HasPrefix(arg, appLinkPrefix) || strings.HasPrefix(arg, webLinkPrefix) {
		return Arrival{Kind: ArrivalLink, Value: arg}, true
	}
	if _, _, _, ok := parseCollectionURL(arg); ok {
		return Arrival{Kind: ArrivalLink, Value: arg}, true
	}
	arg = fileURLPath(arg)
	if !strings.EqualFold(filepath.Ext(arg), ".mortar") {
		return Arrival{}, false
	}
	if info, err := os.Stat(arg); err != nil || !info.Mode().IsRegular() {
		return Arrival{}, false
	}
	return Arrival{Kind: ArrivalFile, Value: arg}, true
}

// LaunchDir is the folder the user launched Mortar from when that is not this process's working folder: an AppImage's
// AppRun changes into the bundle so WebKitGTK finds its helpers, and records the user's folder in OWD.
func LaunchDir() string {
	if os.Getenv("APPDIR") == "" {
		return ""
	}
	return os.Getenv("OWD")
}

// InDir resolves the relative paths among a second launch's arguments against that launch's working folder, which
// is not this process's.
func InDir(args []string, dir string) []string {
	out := slices.Clone(args)
	for i, a := range out {
		if dir != "" && a != "" && !strings.Contains(a, "://") && !filepath.IsAbs(a) {
			out[i] = filepath.Join(dir, a)
		}
	}
	return out
}

// Receive picks the share links and .mortar files out of launch arguments, from a cold start or a second launch,
// and hands each to the window's import dialog. It reports whether there were any.
//
//wails:ignore
func (s *Service) Receive(args []string) bool {
	found := false
	for _, arg := range args {
		a, ok := classify(arg)
		if !ok {
			continue
		}
		found = true
		s.mu.Lock()
		s.nextID++
		a.ID = s.nextID
		s.inbox = append(s.inbox, a)
		s.mu.Unlock()
		s.emit(ArrivedEvent, a)
	}
	return found
}

// Inbox returns what arrived before the window listened, once.
func (s *Service) Inbox() []Arrival {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Arrival{}, s.inbox...)
	s.inbox = nil
	return out
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
