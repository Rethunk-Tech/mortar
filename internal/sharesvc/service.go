// Package sharesvc is the window's side of sharing: it builds the links and .mortar files of a profile, turns a
// link or file into a preview that resolves every mod before anything downloads, and on the user's word creates
// the profile and fills the download queue.
package sharesvc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Rethunk-AI/mortar/internal/fsx"
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
	game    string
	preview Preview
	notes   string
	configs []share.Config
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
	if err := fsx.WriteFile(dest, buf.Bytes(), filePerm); err != nil {
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
	r := &resolver{
		meta: s.d.Meta, files: s.d.Files, signedIn: s.d.SignedIn(), premium: s.d.Premium(), env: s.d.Env(game),
	}
	if profileID != "" {
		p, err := s.find(game, profileID)
		if err != nil {
			return Preview{}, err
		}
		r.target = p.Entries
		if r.installed, err = s.d.Profiles.Installed(game, profileID); err != nil {
			return Preview{}, err
		}
	}
	mods, probs := r.resolve(ctx, shared.Entries)
	pv := Preview{
		Name: shared.Name, Notes: notes, Settings: len(configs), Mods: mods, Problems: probs,
		SignedIn: r.signedIn, Premium: r.premium,
	}
	s.mu.Lock()
	s.current = &session{game: game, preview: pv, notes: notes, configs: configs}
	s.mu.Unlock()
	return pv, nil
}

// Discard forgets the preview; closing the import dialog leaves nothing behind.
func (s *Service) Discard() {
	s.mu.Lock()
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

// Import queues everything the preview lists as available, except the excluded keys, into profileID, or into a
// new profile named as the share when profileID is empty. Mods the preview marked unavailable are listed in the
// new profile's notes. A .mortar file's config files are written once their mods are installed.
func (s *Service) Import(game, profileID string, exclude []string) (Result, error) {
	s.mu.Lock()
	cur := s.current
	s.mu.Unlock()
	if cur == nil || cur.game != game {
		return Result{}, ErrNoPreview
	}
	var reqs []queue.Request
	var wanted []wantedFile
	for _, m := range cur.preview.Mods {
		if slices.Contains(exclude, m.Key) || m.State == StateInstalled || m.State == StateUnavailable {
			continue
		}
		reqs = append(reqs, requestFor(game, "", m))
		wanted = append(wanted, wantedOf(m))
	}
	if slices.ContainsFunc(reqs, func(r queue.Request) bool { return r.Repo == "" }) && !s.d.SignedIn() {
		return Result{}, ErrSignedOut
	}
	var res Result
	created := profileID == ""
	if created {
		p, err := s.d.Profiles.Create(game, cur.preview.Name)
		if err != nil {
			return Result{}, err
		}
		notes := joinNotes(cur.notes, unavailableNote(cur.preview.Mods))
		if utf8.RuneCountInString(notes) > profile.MaxNotes {
			notes = string([]rune(notes)[:profile.MaxNotes])
		}
		if notes != "" {
			if p, err = s.d.Profiles.SetNotes(game, p.ID, notes); err != nil {
				return Result{}, err
			}
		}
		res.Profile = p
		profileID = p.ID
	} else {
		p, err := s.find(game, profileID)
		if err != nil {
			return Result{}, err
		}
		res.Profile = p
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
	if len(cur.configs) > 0 && len(reqs) > 0 {
		s.mu.Lock()
		s.pending = append(s.pending, &pending{Game: game, Profile: profileID, Wanted: wanted, Configs: cur.configs})
		s.mu.Unlock()
		s.savePending()
	}
	s.Discard()
	return res, nil
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
	// seen counts the finished downloads already applied; it starts over with the process.
	seen int
}

func (p *pending) wants(e profile.Entry) bool {
	return slices.ContainsFunc(p.Wanted, func(w wantedFile) bool { return w.entry(e) })
}

func (s *Service) queueChanged(st queue.State) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	defer s.savePending()
	s.mu.Lock()
	todo := slices.Clone(s.pending)
	s.mu.Unlock()
	for _, p := range todo {
		done, open := 0, 0
		for _, it := range st.Items {
			if it.Game != p.Game || it.Profile != p.Profile {
				continue
			}
			if it.State == queue.StateDone && slices.ContainsFunc(p.Wanted, func(w wantedFile) bool { return w.item(it) }) {
				done++
			}
			if !slices.Contains([]string{queue.StateDone, queue.StateSkipped, queue.StateCancelled}, it.State) {
				open++
			}
		}
		if done > p.seen {
			p.seen = done
			s.apply(p)
		}
		if open == 0 || len(p.Configs) == 0 {
			s.mu.Lock()
			s.pending = slices.DeleteFunc(s.pending, func(x *pending) bool { return x == p })
			s.mu.Unlock()
		}
	}
}

// apply writes the config files whose mods are installed, and keeps the rest for later. While the game runs the
// profile it waits for the next change.
func (s *Service) apply(p *pending) {
	if r := s.d.Profiles.Running; r != nil && r(p.Game, p.Profile) {
		p.seen = 0
		return
	}
	prof, err := s.find(p.Game, p.Profile)
	if err != nil {
		return
	}
	modsDir, err := s.d.Profiles.ModsDir(p.Game, p.Profile)
	if err != nil {
		return
	}
	entries := slices.DeleteFunc(slices.Clone(prof.Entries), func(e profile.Entry) bool { return !p.wants(e) })
	written, err := share.Apply(modsDir, entries, p.Configs)
	if err != nil {
		p.seen = 0
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

// classify says whether a launch argument is a share link or a .mortar file. It looks no further than the form:
// the import dialog parses and previews it, and refuses what is not a share.
func classify(arg string) (Arrival, bool) {
	if strings.HasPrefix(arg, appLinkPrefix) || strings.HasPrefix(arg, webLinkPrefix) {
		return Arrival{Kind: ArrivalLink, Value: arg}, true
	}
	if rest, ok := strings.CutPrefix(arg, fileScheme); ok {
		if u, err := url.Parse(fileScheme + rest); err == nil {
			arg = u.Path
		}
	}
	if !strings.EqualFold(filepath.Ext(arg), ".mortar") {
		return Arrival{}, false
	}
	if info, err := os.Stat(arg); err != nil || !info.Mode().IsRegular() {
		return Arrival{}, false
	}
	return Arrival{Kind: ArrivalFile, Value: arg}, true
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
