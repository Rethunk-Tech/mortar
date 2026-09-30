// Package nxmsvc turns nxm:// links into pending downloads and controls whether Mortar handles the scheme.
package nxmsvc

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/nxm"
	"github.com/Rethunk-AI/mortar/internal/settings"
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
	Game    string   `json:"game"`
	Profile string   `json:"profile"`
}

const assignedBuffer = 64

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

	mu         sync.Mutex
	nextID     int
	arrivals   []Arrival
	rejections []Rejection
}

func NewService(store *settings.Store, handler nxm.Handler) *Service {
	return &Service{store: store, handler: handler, now: time.Now, Assigned: make(chan Assignment, assignedBuffer)}
}

func (s *Service) emit(name string, data any) {
	if s.App != nil {
		s.App.Event.Emit(name, data)
	}
}

// Receive reads every nxm:// link among args: an accepted one waits for a profile, a refused one is reported. It
// reports whether there was any link.
func (s *Service) Receive(args []string) bool {
	found := false
	for _, arg := range args {
		if !nxm.IsLink(arg) {
			continue
		}
		found = true
		link, err := nxm.Parse(arg, s.store.Get().NexusUserID, s.now())
		if err == nil && s.Route != nil && s.Route(link) {
			continue
		}
		s.mu.Lock()
		s.nextID++
		id := s.nextID
		if re, ok := errors.AsType[*nxm.RejectError](err); ok {
			r := Rejection{ID: id, Reason: re.Reason}
			s.rejections = append(s.rejections, r)
			s.mu.Unlock()
			s.emit(RejectedEvent, r)
			continue
		}
		a := Arrival{ID: id, Link: link}
		s.arrivals = append(s.arrivals, a)
		s.mu.Unlock()
		s.emit(ArrivedEvent, a)
	}
	return found
}

// Inbox returns the waiting arrivals and the refusals not yet shown; refusals are handed over once.
func (s *Service) Inbox() Inbox {
	s.mu.Lock()
	defer s.mu.Unlock()
	in := Inbox{Arrivals: append([]Arrival{}, s.arrivals...), Rejections: append([]Rejection{}, s.rejections...)}
	s.rejections = nil
	return in
}

func (s *Service) take(id int) (Arrival, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.arrivals {
		if a.ID == id {
			s.arrivals = append(s.arrivals[:i], s.arrivals[i+1:]...)
			return a, true
		}
	}
	return Arrival{}, false
}

// Assign hands the arrival to the download queue for the profile.
func (s *Service) Assign(id int, game, profile string) error {
	if game == "" || profile == "" {
		return errors.New("choose a profile for the download")
	}
	a, ok := s.take(id)
	if !ok {
		return fmt.Errorf("link %d is no longer waiting", id)
	}
	select {
	case s.Assigned <- Assignment{Link: a.Link, Game: game, Profile: profile}:
		return nil
	default:
		return errors.New("too many downloads are waiting")
	}
}

// Ignore drops the arrival.
func (s *Service) Ignore(id int) { s.take(id) }

// Owner names the app that handles nxm links now, or is empty when none does or Mortar already does.
func (s *Service) Owner() (string, error) {
	o, err := s.handler.Owner()
	if err != nil || o.Mine {
		return "", err
	}
	return o.Name, nil
}

// Enable registers Mortar for nxm links and records the owner it replaces.
func (s *Service) Enable() error {
	o, err := s.handler.Owner()
	if err != nil {
		return err
	}
	previous := s.store.Get().NxmPrevious
	if !o.Mine {
		previous = o.ID
	}
	if err := s.handler.Register(); err != nil {
		return err
	}
	return s.record(func(v *settings.Settings) { v.NxmHandled, v.NxmPrevious, v.NxmAsked = true, previous, true })
}

// Disable gives nxm links back to the recorded owner.
func (s *Service) Disable() error {
	if err := s.handler.Restore(s.store.Get().NxmPrevious); err != nil {
		return err
	}
	return s.record(func(v *settings.Settings) { v.NxmHandled, v.NxmPrevious = false, "" })
}

// RegisterLinks makes Mortar the app for mortar:// links and .mortar files; the window calls it once first run is
// done. Running it again is harmless.
func (s *Service) RegisterLinks() error { return s.handler.RegisterLinks() }

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
