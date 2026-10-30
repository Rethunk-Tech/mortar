// Package nxmsvc turns nxm:// links into pending downloads and controls whether Mortar handles the scheme.
package nxmsvc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"

	tstore "github.com/Rethunk-Tech/go-thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/nxm"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/source"
	_ "github.com/Rethunk-Tech/mortar/internal/source/all"
	"github.com/Rethunk-Tech/mortar/internal/source/curseforge"
	"github.com/Rethunk-Tech/mortar/internal/source/vsmoddb"
	"github.com/Rethunk-Tech/mortar/internal/winhost"
)

// Events the window listens to.
const (
	ArrivedEvent  = "nxm:arrived"
	RejectedEvent = "nxm:rejected"
)

// Arrival is an accepted link waiting for the user to say which profile it is for.
type Arrival struct {
	ID   int               `json:"id"`
	Link nexusmods.NXMLink `json:"link"`
	// Package and Version name the Thunderstore package of a ror2mm link, whose Link is empty.
	Package string `json:"package,omitempty"`
	Version string `json:"version,omitempty"`
	// Source is the site of a Package other than Thunderstore ("curseforge", "vsmoddb"), where Package is the project
	// id and File the file the link names, or Version the release (vsmoddb).
	Source string `json:"source,omitempty"`
	File   int    `json:"file,omitempty"`
	// Name is the project's name on its site, when a Source link's project could be read.
	Name string `json:"name,omitempty"`
	// Games are the games the package's site serves: the link names none, so the window picks from these.
	Games []string `json:"games,omitempty"`
	// raw is the link as it arrived, which Ignore forgets so the same link is taken again.
	raw string
}

// Rejection is a refused link; Reason is one of nxm.Reason*.
type Rejection struct {
	ID     int    `json:"id"`
	Reason string `json:"reason"`
	// Name is the mod the link was for, when the reason is about that mod.
	Name string `json:"name,omitempty"`
}

// Inbox is what arrived before the window listened, or is still unanswered.
type Inbox struct {
	Arrivals   []Arrival   `json:"arrivals"`
	Rejections []Rejection `json:"rejections"`
}

// Assignment is a link the user matched to a profile. The download queue takes these from Service.Assigned.
type Assignment struct {
	Link    nexusmods.NXMLink `json:"link"`
	Package string            `json:"package,omitempty"`
	Version string            `json:"version,omitempty"`
	Source  string            `json:"source,omitempty"`
	File    int               `json:"file,omitempty"`
	Game    string            `json:"game"`
	Profile string            `json:"profile"`
}

const assignedBuffer = 64

// duplicateWindow is how long a link without a readable expiry is taken only once. With the browser extension a
// link can arrive more than once: from the page's own launch, from the extension, and again from every tab still
// open when the extension is reinstalled. A link with an expiry is taken once until it expires.
const duplicateWindow = 10 * time.Minute

// Service receives links and switches the system handler.
type Service struct {
	store   *settings.Store
	handler nxm.Handler
	now     func() time.Time
	// App is set after application.New so arrivals can reach the window.
	App winhost.Host
	// CurseForgeProject reads a project's name and the site's game id; nil asks CurseForge.
	CurseForgeProject func(ctx context.Context, id int) (curseforge.Project, error)
	// VSModDBRelease reads the Mod DB mod and release an install link names; nil asks the Mod DB.
	VSModDBRelease func(ctx context.Context, id, version string) (vsmoddb.Resolved, error)
	// Assigned carries links the user matched to a profile.
	Assigned chan Assignment
	// Route is offered every accepted link first and reports whether the download queue was waiting for it; nil
	// means never. A routed link never becomes an arrival.
	Route func(nexusmods.NXMLink) bool

	mu sync.Mutex
	// recent maps each link taken to when it may be taken again.
	recent     map[string]time.Time
	nextID     int
	arrivals   []Arrival
	rejections []Rejection
}

func NewService(store *settings.Store, handler nxm.Handler) *Service {
	cur := store.Get()
	for _, id := range optInSources {
		if on := *handleLinksOf(&cur, id); on != nil {
			source.SetHandleLink(id, *on)
		}
	}
	return &Service{store: store, handler: handler, now: time.Now, Assigned: make(chan Assignment, assignedBuffer)}
}

func (s *Service) emit(name string, data any) {
	if s.App != nil {
		s.App.Emit(name, data)
	}
}

// Receive reads every link among args in a scheme a source claims (nxm://): an accepted one waits for a profile, a refused one is reported. It
// reports whether there was any link.
//
//wails:ignore
func (s *Service) Receive(args []string) bool {
	found := false
	for _, arg := range args {
		if !source.IsLink(arg) {
			continue
		}
		found = true
		if s.duplicate(arg) {
			log.Printf("nxm: duplicate link ignored")
			if link, err := nexusmods.ParseNXM(arg, s.store.Get().NexusUserID, s.now(), game.ByNexusDomain); err == nil {
				s.mu.Lock()
				var waiting *Arrival
				for _, arrival := range s.arrivals {
					if arrival.Link == link {
						duplicate := arrival
						waiting = &duplicate
						break
					}
				}
				s.mu.Unlock()
				if waiting != nil {
					s.emit(ArrivedEvent, *waiting)
				}
			}
			continue
		}
		if ref, err := tstore.ParseLink(arg); err == nil {
			s.mu.Lock()
			s.nextID++
			a := Arrival{ID: s.nextID, Package: ref.Namespace + "-" + ref.Name, Version: ref.Version, Games: gamesWith("thunderstore"), raw: arg}
			log.Printf("nxm: link %d arrived: package %s", a.ID, a.Package)
			s.arrivals = append(s.arrivals, a)
			s.mu.Unlock()
			s.emit(ArrivedEvent, a)
			continue
		}
		if ref, err := curseforge.ParseLink(arg); err == nil {
			go s.receiveCurseForge(arg, ref)
			continue
		}
		if ref, err := vsmoddb.ParseLink(arg); err == nil {
			go s.receiveVSModDB(arg, ref)
			continue
		}
		if domain, gerr := nexusmods.NXMDomain(arg); gerr == nil && !isMortarGame(domain) {
			cur := s.store.Get()
			if previous := cur.NxmPreviousHandlers[linkScheme(arg)]; previous != "" && cur.RedirectOtherGames() {
				err := s.handler.ForwardOther(arg, previous)
				if err == nil {
					log.Printf("nxm: %s link forwarded to the previous handler", domain)
					continue
				}
				log.Printf("nxm: forward %s link: %v", domain, err)
			}
		}
		link, err := nexusmods.ParseNXM(arg, s.store.Get().NexusUserID, s.now(), game.ByNexusDomain)
		if err == nil && s.Route != nil && s.Route(link) {
			log.Printf("nxm: mod %d file %d resumed a waiting download", link.ModID, link.FileID)
			continue
		}
		s.mu.Lock()
		s.nextID++
		id := s.nextID
		if re, ok := errors.AsType[*nexusmods.RejectError](err); ok {
			s.mu.Unlock()
			s.forget(arg)
			s.mu.Lock()
			r := Rejection{ID: id, Reason: re.Reason}
			log.Printf("nxm: link %d rejected: %s", id, re.Reason)
			s.rejections = append(s.rejections, r)
			s.mu.Unlock()
			s.emit(RejectedEvent, r)
			continue
		}
		a := Arrival{ID: id, Link: link, raw: arg}
		log.Printf("nxm: link %d arrived: mod %d file %d", id, link.ModID, link.FileID)
		s.arrivals = append(s.arrivals, a)
		s.mu.Unlock()
		s.emit(ArrivedEvent, a)
	}
	return found
}

// curseforgeLookup bounds the wait for the project a link names.
const curseforgeLookup = 10 * time.Second

// receiveCurseForge asks the site which game the link's project is for, since the link names none. A project of a
// game Mortar does not manage goes back to the app that had these links; when the site cannot be asked, the link
// waits for any game CurseForge serves.
func (s *Service) receiveCurseForge(raw string, ref curseforge.Ref) {
	project := s.CurseForgeProject
	if project == nil {
		project = curseforge.Driver{}.Project
	}
	ctx, cancel := context.WithTimeout(context.Background(), curseforgeLookup)
	found, err := project(ctx, ref.Project)
	name, gameID := found.Name, found.GameID
	cancel()
	games := gamesWith("curseforge")
	if err != nil {
		log.Printf("nxm: curseforge project %d not read: %v", ref.Project, err)
	} else {
		games = nil
		for _, g := range game.Catalog() {
			if slices.ContainsFunc(g.Sources, func(src components.GameSource) bool {
				return src.ID == "curseforge" && src.GameID == gameID
			}) {
				games = append(games, g.ID)
			}
		}
	}
	s.mu.Lock()
	s.nextID++
	id := s.nextID
	s.mu.Unlock()
	if len(games) == 0 {
		if previous := s.store.Get().NxmPreviousHandlers[linkScheme(raw)]; previous != "" {
			if err := s.handler.ForwardOther(raw, previous); err == nil {
				log.Printf("nxm: curseforge project %d forwarded to the previous handler", ref.Project)
				return
			} else {
				log.Printf("nxm: forward curseforge project %d: %v", ref.Project, err)
			}
		}
		s.forget(raw)
		r := Rejection{ID: id, Reason: nexusmods.ReasonGame}
		log.Printf("nxm: link %d rejected: %s", id, r.Reason)
		s.mu.Lock()
		s.rejections = append(s.rejections, r)
		s.mu.Unlock()
		s.emit(RejectedEvent, r)
		return
	}
	if found.Restricted {
		// Mortar may not fetch this file, so the link is refused here instead of failing in the queue.
		s.forget(raw)
		r := Rejection{ID: id, Reason: nxm.ReasonRestricted, Name: name}
		log.Printf("nxm: link %d rejected: %s", id, r.Reason)
		s.mu.Lock()
		s.rejections = append(s.rejections, r)
		s.mu.Unlock()
		s.emit(RejectedEvent, r)
		return
	}
	a := Arrival{ID: id, Source: "curseforge", Package: strconv.Itoa(ref.Project), Name: name, File: ref.File, Games: games, raw: raw}
	log.Printf("nxm: link %d arrived: curseforge project %s file %d", a.ID, a.Package, a.File)
	s.mu.Lock()
	s.arrivals = append(s.arrivals, a)
	s.mu.Unlock()
	s.emit(ArrivedEvent, a)
}

// receiveVSModDB reads the Mod DB's ids for the mod a link names, so the install lands under the same key as one from
// Browse; when the site cannot be asked, the link's own mod id stands in, which the Mod DB also answers to.
func (s *Service) receiveVSModDB(raw string, ref vsmoddb.Ref) {
	release := s.VSModDBRelease
	if release == nil {
		release = vsmoddb.Driver{}.Resolve
	}
	ctx, cancel := context.WithTimeout(context.Background(), curseforgeLookup)
	found, err := release(ctx, ref.ModID, ref.Version)
	cancel()
	a := Arrival{Source: "vsmoddb", Package: ref.ModID, Version: ref.Version, Name: ref.ModID, Games: gamesWith("vsmoddb"), raw: raw}
	if err != nil {
		log.Printf("nxm: mod db mod %s not read: %v", ref.ModID, err)
	} else {
		a.Package, a.Name = found.ID, found.Name
	}
	s.mu.Lock()
	s.nextID++
	a.ID = s.nextID
	s.arrivals = append(s.arrivals, a)
	s.mu.Unlock()
	log.Printf("nxm: link %d arrived: mod db mod %s %s", a.ID, a.Package, a.Version)
	s.emit(ArrivedEvent, a)
}

// gamesWith lists the games the source serves.
func gamesWith(sourceID string) []string {
	var out []string
	for _, g := range game.Catalog() {
		if slices.ContainsFunc(g.Sources, func(src components.GameSource) bool { return src.ID == sourceID }) {
			out = append(out, g.ID)
		}
	}
	return out
}

func isMortarGame(domain string) bool {
	_, ok := game.ByNexusDomain(domain)
	return ok
}

func (s *Service) forget(link string) {
	s.mu.Lock()
	delete(s.recent, link)
	s.mu.Unlock()
}

// duplicate records link and reports whether it already arrived and has not expired since.
func (s *Service) duplicate(link string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for l, until := range s.recent {
		if now.After(until) {
			delete(s.recent, l)
		}
	}
	if _, ok := s.recent[link]; ok {
		return true
	}
	if s.recent == nil {
		s.recent = map[string]time.Time{}
	}
	until := now.Add(duplicateWindow)
	if u, err := url.Parse(link); err == nil {
		if exp, err := strconv.ParseInt(u.Query().Get("expires"), 10, 64); err == nil && time.Unix(exp, 0).After(until) {
			until = time.Unix(exp, 0)
		}
	}
	s.recent[link] = until
	return false
}

// Inbox returns the waiting arrivals and the refusals not yet shown; refusals are handed over once.
func (s *Service) Inbox() Inbox {
	s.mu.Lock()
	defer s.mu.Unlock()
	in := Inbox{Arrivals: append([]Arrival{}, s.arrivals...), Rejections: append([]Rejection{}, s.rejections...)}
	s.rejections = nil
	return in
}

// Assign hands the arrival to the download queue for the profile.
func (s *Service) Assign(id int, game, profile string) error {
	if game == "" || profile == "" {
		return errors.New("choose a profile for the download")
	}
	// The arrival leaves the inbox only once the queue took it, so a full queue leaves it there to assign again.
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.arrivals, func(a Arrival) bool { return a.ID == id })
	if i < 0 {
		return fmt.Errorf("link %d is no longer waiting", id)
	}
	select {
	case s.Assigned <- Assignment{
		Link: s.arrivals[i].Link, Package: s.arrivals[i].Package, Version: s.arrivals[i].Version,
		Source: s.arrivals[i].Source, File: s.arrivals[i].File, Game: game, Profile: profile,
	}:
		log.Printf("nxm: link %d assigned to profile %s", id, profile)
		s.arrivals = slices.Delete(s.arrivals, i, i+1)
		return nil
	default:
		log.Printf("nxm: link %d not assigned: download queue full", id)
		return errors.New("too many downloads are waiting")
	}
}

// Ignore drops the arrival.
func (s *Service) Ignore(id int) {
	s.mu.Lock()
	raw := ""
	s.arrivals = slices.DeleteFunc(s.arrivals, func(a Arrival) bool {
		if a.ID == id {
			raw = a.raw
			return true
		}
		return false
	})
	s.mu.Unlock()
	s.forget(raw)
	log.Printf("nxm: link %d ignored", id)
}

// Owner names the app that handles nxm links now, or is empty when none does or Mortar already does.
func (s *Service) Owner() (string, error) {
	o, err := s.handler.Owner(nxmScheme)
	if err != nil || o.Mine {
		return "", err
	}
	return o.Name, nil
}

// nxmScheme is the scheme whose previous owner is shown to the user.
const nxmScheme = "nxm"

func linkScheme(link string) string {
	scheme, _, _ := strings.Cut(link, ":")
	return strings.ToLower(scheme)
}

// Enable registers Mortar for nxm links and records the owner it replaces.
func (s *Service) Enable() error {
	previous := maps.Clone(s.store.Get().NxmPreviousHandlers)
	if previous == nil {
		previous = map[string]string{}
	}
	prevName := s.store.Get().NxmPreviousName
	for _, scheme := range nxm.Schemes() {
		o, err := s.handler.Owner(scheme)
		if err != nil {
			return err
		}
		if o.Mine {
			continue
		}
		if o.ID == "" {
			delete(previous, scheme)
		} else {
			previous[scheme] = o.ID
		}
		if scheme == nxmScheme {
			prevName = o.Name
		}
	}
	if err := s.handler.Register(); err != nil {
		return err
	}
	return s.record(func(v *settings.Settings) {
		v.NxmHandled, v.NxmPreviousHandlers, v.NxmPreviousName, v.NxmAsked = true, previous, prevName, true
	})
}

// LinkOwner is who the system opens one URL scheme with.
type LinkOwner struct {
	Scheme string `json:"scheme"`
	Name   string `json:"name"`
	ID     string `json:"id"`
	Mine   bool   `json:"mine"`
}

// ListOwners reports the current handler of every source scheme and of mortar://. One scheme's error is a row, not a
// failed list, so a desktop without a MIME database still names the rest.
func (s *Service) ListOwners() []LinkOwner {
	seen := map[string]bool{}
	var schemes []string
	add := func(scheme string) {
		if scheme == "" || seen[scheme] {
			return
		}
		seen[scheme] = true
		schemes = append(schemes, scheme)
	}
	for _, src := range source.All() {
		for _, scheme := range source.SchemesOf(src.ID()) {
			add(scheme)
		}
	}
	add("mortar")
	out := make([]LinkOwner, 0, len(schemes))
	for _, scheme := range schemes {
		row := LinkOwner{Scheme: scheme}
		o, err := s.handler.Owner(scheme)
		if err != nil {
			row.Name = err.Error()
		} else {
			row.Name, row.ID, row.Mine = o.Name, o.ID, o.Mine
		}
		out = append(out, row)
	}
	return out
}

// LinkSource is a source whose links Mortar handles only when the user chooses, with its scheme.
type LinkSource struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Scheme  string `json:"scheme"`
	Handled bool   `json:"handled"`
}

// LinkSources lists the opt-in sources that have a link scheme (Thunderstore's ror2mm), with whether Mortar handles
// each now; Nexus's nxm has its own switch.
func (s *Service) LinkSources() []LinkSource {
	out := []LinkSource{}
	for _, src := range source.All() {
		schemes := source.SchemesOf(src.ID())
		if !hasOptInSetting(src.ID()) || !source.OptsIn(src.ID()) || len(schemes) == 0 {
			continue
		}
		out = append(out, LinkSource{ID: src.ID(), Name: src.Name(), Scheme: schemes[0], Handled: source.Claims(src.ID())})
	}
	return out
}

// EnableSource makes Mortar handle the links of one source. An opt-in source (Thunderstore's ror2mm) is claimed and
// recorded in its own setting; Nexus is the nxm path of Enable.
func (s *Service) EnableSource(id string) error {
	if !source.OptsIn(id) {
		return s.Enable()
	}
	if !hasOptInSetting(id) {
		return fmt.Errorf("source %q has no link scheme to handle", id)
	}
	return ClaimLinks(s.store, s.handler, id)
}

// DisableSource gives one source's links back to their recorded owners and leaves the other sources registered.
func (s *Service) DisableSource(id string) error {
	if !source.OptsIn(id) {
		return s.Disable()
	}
	if !hasOptInSetting(id) {
		return fmt.Errorf("source %q has no link scheme to handle", id)
	}
	schemes := source.SchemesOf(id)
	previous := maps.Clone(s.store.Get().NxmPreviousHandlers)
	source.SetHandleLink(id, false)
	if err := s.handler.Release(schemes, previous); err != nil {
		source.SetHandleLink(id, true)
		return err
	}
	for _, scheme := range schemes {
		delete(previous, scheme)
	}
	return s.record(func(v *settings.Settings) {
		v.NxmPreviousHandlers = previous
		setHandleLinks(v, id, false)
	})
}

// ClaimLinks makes Mortar handle one source's links (the installer's checkboxes) and leaves the other sources'
// schemes as they are.
func ClaimLinks(store *settings.Store, handler nxm.Handler, id string) error {
	if id == "nexus" {
		return NewService(store, handler).Enable()
	}
	if !hasOptInSetting(id) {
		return fmt.Errorf("source %q has no link scheme to handle", id)
	}
	s := NewService(store, handler)
	previous := maps.Clone(store.Get().NxmPreviousHandlers)
	if previous == nil {
		previous = map[string]string{}
	}
	schemes := source.SchemesOf(id)
	for _, scheme := range schemes {
		o, err := handler.Owner(scheme)
		if err != nil {
			return err
		}
		if !o.Mine && o.ID != "" {
			previous[scheme] = o.ID
		}
	}
	source.SetHandleLink(id, true)
	if err := handler.RegisterSchemes(schemes); err != nil {
		return err
	}
	return s.record(func(v *settings.Settings) {
		v.NxmPreviousHandlers = previous
		setHandleLinks(v, id, true)
	})
}

// optInSources are the sources whose link choice is stored.
var optInSources = []string{"thunderstore", "curseforge", "vsmoddb"}

func hasOptInSetting(id string) bool { return slices.Contains(optInSources, id) }

// handleLinksOf is the stored link choice of an opt-in source.
func handleLinksOf(v *settings.Settings, id string) **bool {
	switch id {
	case "curseforge":
		return &v.CurseForgeHandleLinks
	case "vsmoddb":
		return &v.VSModDBHandleLinks
	}
	return &v.ThunderstoreHandleLinks
}

func setHandleLinks(v *settings.Settings, id string, on bool) {
	if hasOptInSetting(id) {
		*handleLinksOf(v, id) = &on
	}
}

// Disable gives nxm links back to the recorded owner. Another source's claimed scheme (ror2mm) stays registered.
func (s *Service) Disable() error {
	nexus := source.SchemesOf("nexus")
	if len(source.Schemes()) == len(nexus) {
		return release(s)
	}
	previous := maps.Clone(s.store.Get().NxmPreviousHandlers)
	if err := s.handler.Release(nexus, previous); err != nil {
		return err
	}
	for _, scheme := range nexus {
		delete(previous, scheme)
	}
	return s.record(func(v *settings.Settings) {
		v.NxmHandled, v.NxmPreviousHandlers, v.NxmPreviousName = false, previous, ""
	})
}

// ReleaseLinks restores the nxm handler recorded in settings, the same path Settings uses when the toggle is turned off.
func ReleaseLinks(store *settings.Store, handler nxm.Handler) error {
	return release(NewService(store, handler))
}

func release(s *Service) error {
	if err := s.handler.Restore(s.store.Get().NxmPreviousHandlers); err != nil {
		return err
	}
	return s.record(func(v *settings.Settings) { v.NxmHandled, v.NxmPreviousHandlers, v.NxmPreviousName = false, nil, "" })
}

// RegisterLinks makes Mortar the app for mortar:// links and .mortar files; the window calls it once first run is
// done. Running it again is harmless.
func (s *Service) RegisterLinks() error { return s.handler.RegisterLinks() }

// NotificationIcon is the PNG to attach to a desktop notification, or empty when it has not been installed.
func (s *Service) NotificationIcon() string {
	type withIcon interface{ NotificationIcon() string }
	if h, ok := s.handler.(withIcon); ok {
		return h.NotificationIcon()
	}
	return ""
}

// Takeover is an app that took a source's links over while Mortar is set to handle them.
type Takeover struct {
	Source string `json:"source"`
	Name   string `json:"name"`
}

// claimed maps each source whose links Mortar is set to handle to its schemes.
func claimed(v settings.Settings) map[string][]string {
	out := map[string][]string{}
	if v.NxmHandled {
		out["nexus"] = []string{nxmScheme}
	}
	for _, id := range optInSources {
		if on := *handleLinksOf(&v, id); on != nil && *on {
			out[id] = source.SchemesOf(id)
		}
	}
	return out
}

// Taken lists the sources whose links another app took over while Mortar is set to handle them (Vortex and
// r2modman re-register themselves whenever they start).
func (s *Service) Taken() ([]Takeover, error) {
	byID := claimed(s.store.Get())
	out := []Takeover{}
	for _, id := range append([]string{"nexus"}, optInSources...) {
		for _, scheme := range byID[id] {
			o, err := s.handler.Owner(scheme)
			if err != nil {
				return nil, err
			}
			if !o.Mine && o.ID != "" {
				out = append(out, Takeover{Source: id, Name: o.Name})
				break
			}
		}
	}
	return out, nil
}

// TakeBack registers Mortar again for every scheme it is set to handle, keeping what each replaces for give-back.
func (s *Service) TakeBack() error {
	byID := claimed(s.store.Get())
	previous := maps.Clone(s.store.Get().NxmPreviousHandlers)
	if previous == nil {
		previous = map[string]string{}
	}
	var schemes []string
	for _, sc := range byID {
		for _, scheme := range sc {
			o, err := s.handler.Owner(scheme)
			if err != nil {
				return err
			}
			if !o.Mine && o.ID != "" {
				previous[scheme] = o.ID
			}
			schemes = append(schemes, scheme)
		}
	}
	if err := s.handler.RegisterSchemes(schemes); err != nil {
		return err
	}
	return s.record(func(v *settings.Settings) { v.NxmPreviousHandlers = previous })
}

// Yield leaves the given sources' links with the apps that took them: Mortar stops claiming them and changes
// nothing in the system.
func (s *Service) Yield(sources []string) error {
	byID := claimed(s.store.Get())
	return s.record(func(v *settings.Settings) {
		for _, id := range sources {
			for _, scheme := range byID[id] {
				delete(v.NxmPreviousHandlers, scheme)
			}
			switch id {
			case "nexus":
				v.NxmHandled, v.NxmPreviousName = false, ""
			default:
				if hasOptInSetting(id) {
					source.SetHandleLink(id, false)
					setHandleLinks(v, id, false)
				}
			}
		}
	})
}

// DeclineOffer records that the user was asked and said not now.
func (s *Service) DeclineOffer() error {
	return s.record(func(v *settings.Settings) { v.NxmAsked = true })
}

func (s *Service) record(fn func(*settings.Settings)) error {
	next, err := s.store.Update(fn)
	if err != nil {
		return err
	}
	s.emit(settings.ChangedEvent, next)
	return nil
}
