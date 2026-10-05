// Package nxmsvc turns nxm:// links into pending downloads and controls whether Mortar handles the scheme.
package nxmsvc

import (
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

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/nxm"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/source"
	_ "github.com/Rethunk-Tech/mortar/internal/source/all"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Events the window listens to.
const (
	ArrivedEvent  = "nxm:arrived"
	RejectedEvent = "nxm:rejected"
)

// Arrival is an accepted link waiting for the user to say which profile it is for.
type Arrival struct {
	ID   int      `json:"id"`
	Link nxm.Link `json:"link"`
	// Package and Version name the Thunderstore package of a ror2mm link, whose Link is empty.
	Package string `json:"package,omitempty"`
	Version string `json:"version,omitempty"`
	// Games are the games with a Thunderstore community: the link names none, so the window picks from these.
	Games []string `json:"games,omitempty"`
}

// Rejection is a refused link; Reason is one of nxm.Reason*.
type Rejection struct {
	ID     int    `json:"id"`
	Reason string `json:"reason"`
}

// Inbox is what arrived before the window listened, or is still unanswered.
type Inbox struct {
	Arrivals   []Arrival   `json:"arrivals"`
	Rejections []Rejection `json:"rejections"`
}

// Assignment is a link the user matched to a profile. The download queue takes these from Service.Assigned.
type Assignment struct {
	Link    nxm.Link `json:"link"`
	Package string   `json:"package,omitempty"`
	Version string   `json:"version,omitempty"`
	Game    string   `json:"game"`
	Profile string   `json:"profile"`
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
	App *application.App
	// Assigned carries links the user matched to a profile.
	Assigned chan Assignment
	// Route is offered every accepted link first and reports whether the download queue was waiting for it; nil
	// means never. A routed link never becomes an arrival.
	Route func(nxm.Link) bool

	mu sync.Mutex
	// recent maps each link taken to when it may be taken again.
	recent     map[string]time.Time
	nextID     int
	arrivals   []Arrival
	rejections []Rejection
}

func NewService(store *settings.Store, handler nxm.Handler) *Service {
	if on := store.Get().ThunderstoreHandleLinks; on != nil {
		source.SetHandleLink("thunderstore", *on)
	}
	return &Service{store: store, handler: handler, now: time.Now, Assigned: make(chan Assignment, assignedBuffer)}
}

func (s *Service) emit(name string, data any) {
	if s.App != nil {
		s.App.Event.Emit(name, data)
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
			if link, err := nxm.Parse(arg, s.store.Get().NexusUserID, s.now(), game.ByNexusDomain); err == nil {
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
		if ref, err := thunderstore.ParseLink(arg); err == nil {
			s.mu.Lock()
			s.nextID++
			a := Arrival{ID: s.nextID, Package: ref.Namespace + "-" + ref.Name, Version: ref.Version, Games: thunderstoreGames()}
			log.Printf("nxm: link %d arrived: package %s", a.ID, a.Package)
			s.arrivals = append(s.arrivals, a)
			s.mu.Unlock()
			s.emit(ArrivedEvent, a)
			continue
		}
		if domain, gerr := nxm.LinkGame(arg); gerr == nil && !isMortarGame(domain) {
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
		link, err := nxm.Parse(arg, s.store.Get().NexusUserID, s.now(), game.ByNexusDomain)
		if err == nil && s.Route != nil && s.Route(link) {
			log.Printf("nxm: mod %d file %d resumed a waiting download", link.ModID, link.FileID)
			continue
		}
		s.mu.Lock()
		s.nextID++
		id := s.nextID
		if re, ok := errors.AsType[*nxm.RejectError](err); ok {
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
		a := Arrival{ID: id, Link: link}
		log.Printf("nxm: link %d arrived: mod %d file %d", id, link.ModID, link.FileID)
		s.arrivals = append(s.arrivals, a)
		s.mu.Unlock()
		s.emit(ArrivedEvent, a)
	}
	return found
}

func thunderstoreGames() []string {
	var out []string
	for _, g := range game.Catalog() {
		if slices.ContainsFunc(g.Sources, func(src components.GameSource) bool { return src.ID == "thunderstore" }) {
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

func (s *Service) forgetArrival(link nxm.Link) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for raw := range s.recent {
		if parsed, err := nxm.Parse(raw, s.store.Get().NexusUserID, s.now(), game.ByNexusDomain); err == nil && parsed == link {
			delete(s.recent, raw)
		}
	}
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
	case s.Assigned <- Assignment{Link: s.arrivals[i].Link, Package: s.arrivals[i].Package, Version: s.arrivals[i].Version, Game: game, Profile: profile}:
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
	var link nxm.Link
	s.arrivals = slices.DeleteFunc(s.arrivals, func(a Arrival) bool {
		if a.ID == id {
			link = a.Link
			return true
		}
		return false
	})
	s.mu.Unlock()
	if link.ModID != 0 {
		s.forgetArrival(link)
	}
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
	source.SetHandleLink(id, true)
	if err := s.Enable(); err != nil {
		source.SetHandleLink(id, false)
		return err
	}
	return s.record(func(v *settings.Settings) { setHandleLinks(v, id, true) })
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

// hasOptInSetting says whether the source's choice is stored; only Thunderstore's is.
func hasOptInSetting(id string) bool { return id == "thunderstore" }

func setHandleLinks(v *settings.Settings, id string, on bool) {
	if id == "thunderstore" {
		v.ThunderstoreHandleLinks = &on
	}
}

// Disable gives nxm links back to the recorded owner.
func (s *Service) Disable() error {
	return release(s)
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
