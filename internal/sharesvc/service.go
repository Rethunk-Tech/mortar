// Package sharesvc is the window's side of sharing: it builds the links and .mortar files of a profile, turns a
// link or file into a preview that resolves every mod before anything downloads, and on the user's word creates
// the profile and fills the download queue.
package sharesvc

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/queue"
	"github.com/Rethunk-AI/mortar/internal/share"
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
)

const filePerm = 0o644

var (
	// ErrNoPreview means Import was asked for without a preview to act on.
	ErrNoPreview = errors.New("there is nothing to import: open a link or file first")
	// ErrStalePreview means Import named a preview other than the one the service holds.
	ErrStalePreview = errors.New("the preview changed: check it again before importing")
	// ErrSignedOut means the import has Nexus downloads and no account is signed in.
	ErrSignedOut = errors.New("sign in to Nexus Mods before importing mods from Nexus")
)

// Arrival is a link or a .mortar file that reached the app from outside. It is only ever shown in the import
// dialog; nothing is imported by arriving.
type Arrival struct {
	ID    int    `json:"id"`
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Queue is the download queue's part in an import.
type Queue interface {
	Add(reqs []queue.Request) ([]queue.Item, error)
}

// Deps is what the service needs from the rest of the app.
type Deps struct {
	Profiles *profile.Store
	Meta     problems.Meta
	// Files lists a Nexus mod's files; it is only called while SignedIn.
	Files    func(ctx context.Context, modID int) ([]nexus.File, error)
	SignedIn func() bool
	Premium  func() bool
	Env      func(game string) problems.Environment
	Queue    Queue
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
}

// NewService returns the service.
func NewService(d Deps) *Service {
	s := &Service{d: d}
	s.pending = loadPending(d.Dir)
	s.QueueChanged = s.queueChanged
	return s
}

// session is the preview the dialog is showing, kept so Import acts on exactly what was shown.
type session struct {
	id      string
	game    string
	preview Preview
	notes   string
	configs []share.Config
	// target is the profile the preview was resolved against; refs are what it resolved.
	target string
	refs   []share.Ref
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
	i := slices.IndexFunc(all, func(p profile.Profile) bool { return p.ID == id })
	if i < 0 {
		return profile.Profile{}, fmt.Errorf("profile %s not found", id)
	}
	return all[i], nil
}

func entryNames(e profile.Entry, withDisabled bool) []string {
	var names []string
	for _, m := range e.Mods {
		if (withDisabled || !slices.Contains(e.Disabled, m.UniqueID)) && m.Name != "" {
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

// Share builds the link of a profile, and what it holds and leaves out.
func (s *Service) Share(game, profileID string) (Info, error) {
	p, err := s.find(game, profileID)
	if err != nil {
		return Info{}, err
	}
	return describe(p)
}

func describe(p profile.Profile) (Info, error) {
	shared, left, off := share.Collect(p)
	info := Info{Name: p.Name, Limit: DiscordLimit, Count: len(shared.Entries), Groups: []Group{}, LeftOut: []Omitted{}}
	res, err := share.Encode(p)
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
		if e.Source.Kind != profile.KindNexus && e.Source.Kind != profile.KindGitHub || leftKeys[e.Key] || slices.Contains(off, e.Key) {
			continue
		}
		g := groups[e.Source.Kind]
		if g == nil {
			info.Groups = append(info.Groups, Group{Source: e.Source.Kind})
			g = &info.Groups[len(info.Groups)-1]
			groups[e.Source.Kind] = g
		}
		g.Mods = append(g.Mods, entryNames(e, false)...)
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
func (s *Service) SaveFile(game, profileID string) (Saved, error) {
	p, err := s.find(game, profileID)
	if err != nil {
		return Saved{}, err
	}
	modsDir, err := s.d.Profiles.ModsDir(game, profileID)
	if err != nil {
		return Saved{}, err
	}
	d := s.App.Dialog.SaveFile().
		SetFilename(fileNameUnsafe.Replace(p.Name)+".mortar").
		AddFilter("Mortar profile (.mortar)", "*.mortar")
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	dest, err := d.PromptForSingleSelection()
	if err != nil || dest == "" {
		return Saved{Skipped: []string{}}, err
	}
	if !strings.EqualFold(filepath.Ext(dest), ".mortar") {
		dest += ".mortar"
	}
	var buf bytes.Buffer
	skipped, err := share.Write(&buf, p, modsDir)
	if err != nil {
		return Saved{}, err
	}
	if err := datadir.WriteFile(dest, buf.Bytes(), filePerm); err != nil {
		return Saved{}, err
	}
	return Saved{Path: dest, Skipped: append([]string{}, skipped...)}, nil
}

// --- Import ---

// PickFile asks for a .mortar file and returns its path, or "" when the dialog is cancelled.
func (s *Service) PickFile() (string, error) {
	d := s.App.Dialog.OpenFile().
		SetTitle("Open a .mortar file").
		AddFilter("Mortar profile (.mortar)", "*.mortar")
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	return d.PromptForSingleSelection()
}

// ReadClipboard returns the clipboard's text. It is called only when the user presses Paste in the import dialog.
func (s *Service) ReadClipboard() string {
	text, _ := s.App.Clipboard.Text()
	return text
}

// PreviewLink reads a share link, or a bare payload, and resolves what it names against the profile it would join
// (profileID may be empty).
func (s *Service) PreviewLink(ctx context.Context, game, text, profileID string) (Preview, error) {
	shared, err := share.Parse(text)
	if err != nil {
		return Preview{}, err
	}
	return s.preview(ctx, game, shared, "", nil, profileID)
}

// PreviewFile reads a .mortar file and resolves what it names.
func (s *Service) PreviewFile(ctx context.Context, game, file, profileID string) (Preview, error) {
	pv, err := share.Read(file)
	if err != nil {
		return Preview{}, err
	}
	return s.preview(ctx, game, pv.Shared, pv.Notes, pv.Configs, profileID)
}

func (s *Service) preview(ctx context.Context, game string, shared share.Shared, notes string, configs []share.Config, profileID string) (Preview, error) {
	s.mu.Lock()
	s.gen++
	gen := s.gen
	s.mu.Unlock()
	r, err := s.resolverFor(game, profileID)
	if err != nil {
		return Preview{}, err
	}
	mods, probs := r.resolve(ctx, shared.Entries)
	var raw [8]byte
	_, _ = rand.Read(raw[:])
	pv := Preview{
		Session: hex.EncodeToString(raw[:]), Name: shared.Name, Notes: notes, Settings: len(configs), Mods: mods, Problems: probs,
		SignedIn: r.signedIn, Premium: r.premium,
	}
	s.mu.Lock()
	if s.gen == gen {
		s.current = &session{
			id: pv.Session, game: game, preview: pv, notes: notes, configs: configs, target: profileID, refs: shared.Entries,
		}
	}
	s.mu.Unlock()
	return pv, nil
}

// resolverFor resolves against profileID, whose mods count as installed, or against nothing when it is empty.
func (s *Service) resolverFor(game, profileID string) (*resolver, error) {
	r := &resolver{
		meta: s.d.Meta, files: s.d.Files, signedIn: s.d.SignedIn(), premium: s.d.Premium(), env: s.d.Env(game),
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
}

func requestFor(game, profileID string, m Mod) queue.Request {
	kind := queue.KindInstall
	if m.State == StateDependency {
		kind = queue.KindDependency
	}
	r := queue.Request{Kind: kind, Game: game, Profile: profileID, Name: m.Name, Version: m.Version}
	if m.Site == SiteGitHub {
		r.Repo, r.Tag, r.Asset = m.Repo, m.Tag, m.Asset
		return r
	}
	r.ModID, r.FileID = m.ModID, m.FileID
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

// Import queues everything the preview named session lists as available, except the excluded keys, into
// profileID, or into a new profile named as the share (with " (2)" and so on when that name is taken) when profileID
// is empty. Mods the preview marked unavailable are listed in the new profile's notes. A .mortar file's config files
// are written once their mods are installed, except for mods the profile already had, whose config is the user's.
// A preview resolved against another profile than profileID is resolved again, so a new profile made from a preview
// of the open one still gets the mods the open one has.
func (s *Service) Import(ctx context.Context, game, session, profileID string, exclude []string) (res Result, err error) {
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
	mods := cur.preview.Mods
	if profileID != cur.target {
		r, err := s.resolverFor(game, profileID)
		if err != nil {
			return Result{}, err
		}
		mods, _ = r.resolve(ctx, cur.refs)
	}
	var reqs []queue.Request
	var wanted []wantedFile
	for _, m := range mods {
		if slices.Contains(exclude, m.Key) || m.State == StateInstalled || m.State == StateUnavailable {
			continue
		}
		reqs = append(reqs, requestFor(game, "", m))
		wanted = append(wanted, wantedOf(m))
	}
	if slices.ContainsFunc(reqs, func(r queue.Request) bool { return r.Repo == "" }) && !s.d.SignedIn() {
		return Result{}, ErrSignedOut
	}
	configs := cur.configs
	created := profileID == ""
	if created {
		existing, err := s.d.Profiles.List(game)
		if err != nil {
			return Result{}, err
		}
		names := make([]string, len(existing))
		for i, e := range existing {
			names[i] = e.Name
		}
		p, err := s.d.Profiles.Create(game, profile.UniqueName(names, cur.preview.Name))
		if err != nil {
			return Result{}, err
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
		res.Profile = p
		profileID = p.ID
	} else {
		p, err := s.find(game, profileID)
		if err != nil {
			return Result{}, err
		}
		res.Profile = p
		configs = slices.DeleteFunc(slices.Clone(configs), func(c share.Config) bool { return holds(p, c.UniqueID) })
	}
	for i := range reqs {
		reqs[i].Profile = profileID
	}
	if len(reqs) > 0 {
		if _, err := s.d.Queue.Add(reqs); err != nil {
			if created {
				err = errors.Join(err, s.d.Profiles.Delete(game, profileID))
			}
			return Result{}, err
		}
	}
	res.Queued = len(reqs)
	if len(configs) > 0 && len(reqs) > 0 {
		// savePending marshals every pending import, which queueChanged edits under applyMu.
		s.applyMu.Lock()
		s.mu.Lock()
		s.pending = append(s.pending, &pending{Game: game, Profile: profileID, Wanted: wanted, Configs: configs})
		s.mu.Unlock()
		s.savePending()
		s.applyMu.Unlock()
	}
	return res, nil
}

func holds(p profile.Profile, uniqueID string) bool {
	return slices.ContainsFunc(p.Entries, func(e profile.Entry) bool {
		return slices.ContainsFunc(e.Mods, func(m profile.EntryMod) bool { return strings.EqualFold(m.UniqueID, uniqueID) })
	})
}

// --- Config files ---

// wantedFile names a file an import queued, as the profile's entry for it will record its source.
type wantedFile struct {
	ModID  int    `json:"modId"`
	FileID int    `json:"fileId"`
	Repo   string `json:"repo"`
	Tag    string `json:"tag"`
	Asset  string `json:"asset"`
}

func wantedOf(m Mod) wantedFile {
	return wantedFile{ModID: m.ModID, FileID: m.FileID, Repo: m.Repo, Tag: m.Tag, Asset: m.Asset}
}

func (w wantedFile) entry(e profile.Entry) bool {
	if w.Repo != "" {
		return e.Source.Kind == profile.KindGitHub && strings.EqualFold(e.Source.Repo, w.Repo) && (w.Tag == "" || e.Source.Tag == w.Tag) &&
			(w.Asset == "" || e.Source.Asset == w.Asset)
	}
	return e.Source.Kind == profile.KindNexus && e.Source.ModID == w.ModID && e.Source.FileID == w.FileID
}

func (w wantedFile) item(it queue.Item) bool {
	if w.Repo != "" {
		return strings.EqualFold(it.Repo, w.Repo)
	}
	return it.ModID == w.ModID && it.FileID == w.FileID
}

// pending is a .mortar file's config files waiting for the mods they belong to.
type pending struct {
	Game    string         `json:"game"`
	Profile string         `json:"profile"`
	Wanted  []wantedFile   `json:"wanted"`
	Configs []share.Config `json:"configs"`
	// seen is the set of finished downloads the configs were last applied for; it starts over with the process.
	seen string
}

func (p *pending) wants(e profile.Entry) bool {
	return slices.ContainsFunc(p.Wanted, func(w wantedFile) bool { return w.entry(e) })
}

// queueChanged runs on every queue change, progress ticks included, so it saves only when a pending import changed.
func (s *Service) queueChanged(st queue.State) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	s.mu.Lock()
	todo := slices.Clone(s.pending)
	s.mu.Unlock()
	changed := false
	for _, p := range todo {
		var done []string
		open := 0
		for _, it := range st.Items {
			if it.Game != p.Game || it.Profile != p.Profile {
				continue
			}
			if it.State == queue.StateDone && slices.ContainsFunc(p.Wanted, func(w wantedFile) bool { return w.item(it) }) {
				done = append(done, it.ID)
			}
			if !slices.Contains([]string{queue.StateDone, queue.StateSkipped, queue.StateCancelled}, it.State) {
				open++
			}
		}
		// The queue drops old finished items, so the set is compared, not its size.
		slices.Sort(done)
		if seen := strings.Join(done, ","); len(done) > 0 && seen != p.seen && len(p.Configs) > 0 {
			p.seen = seen
			before := len(p.Configs)
			s.apply(p)
			changed = changed || len(p.Configs) != before
		}
		// An apply the running game or a write error held back leaves seen empty, and the import waits.
		if len(p.Configs) == 0 || (open == 0 && (len(done) == 0 || p.seen != "")) {
			s.mu.Lock()
			s.pending = slices.DeleteFunc(s.pending, func(x *pending) bool { return x == p })
			s.mu.Unlock()
			changed = true
		}
	}
	if changed {
		s.savePending()
	}
}

// errRunning stops an apply that found the game running the profile, under the lock a launch takes.
var errRunning = errors.New("the game is running the profile")

// apply writes the config files whose mods are installed, and keeps the rest for later. While the game runs the
// profile it waits for the next change.
func (s *Service) apply(p *pending) {
	var written []string
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
		return slices.ContainsFunc(written, func(id string) bool { return strings.EqualFold(id, c.UniqueID) })
	})
}

// --- Links and files from outside ---

const (
	webLinkPrefix = "https://mortar.rethunk.tech/"
	appLinkPrefix = "mortar://"
	fileScheme    = "file://"
)

// fileURLPath turns a file:// URL into a filesystem path. Windows URLs are /C:/... after parse.
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
	if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return filepath.FromSlash(path)
}

// classify says whether a launch argument is a share link or a .mortar file. It looks no further than the form:
// the import dialog parses and previews it, and refuses what is not a share.
func classify(arg string) (Arrival, bool) {
	if strings.HasPrefix(arg, appLinkPrefix) || strings.HasPrefix(arg, webLinkPrefix) {
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
