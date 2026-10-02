// Package shortcut makes desktop shortcuts that play one profile, and takes the play request such a shortcut sends.
// A shortcut runs Mortar with --play=<game>/<profile>; a running Mortar receives it as a second instance, and a
// closed one starts with it, so the window plays the profile through its usual Play path either way.
package shortcut

import (
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/nxm"
)

const playFlag = "--play="

// RequestedEvent tells the window a shortcut asked to play a profile.
const RequestedEvent = "play:requested"

// Request is one profile a shortcut asked to play.
type Request struct {
	Game    string `json:"game"`
	Profile string `json:"profile"`
}

// Arg is the command-line argument that plays profile of game.
func Arg(game, profile string) string { return playFlag + game + "/" + profile }

// Parse finds a play request among args.
func Parse(args []string) (Request, bool) {
	for _, a := range args {
		rest, ok := strings.CutPrefix(a, playFlag)
		if !ok {
			continue
		}
		game, profile, ok := strings.Cut(rest, "/")
		if ok && validID(game) && validID(profile) {
			return Request{Game: game, Profile: profile}, true
		}
	}
	return Request{}, false
}

func validID(s string) bool {
	return s != "" && !strings.ContainsAny(s, `/\ "'`)
}

// Service holds the latest play request until the window takes it, and makes shortcuts.
type Service struct {
	// Emit is nil in tests that do not watch events.
	Emit    func(name string, data any)
	mu      sync.Mutex
	pending *Request
}

// Receive keeps a play request found in args and tells the window; it reports whether there was one.
func (s *Service) Receive(args []string) bool {
	r, ok := Parse(args)
	if !ok {
		return false
	}
	s.mu.Lock()
	s.pending = &r
	s.mu.Unlock()
	if s.Emit != nil {
		s.Emit(RequestedEvent, r)
	}
	return true
}

// Take returns the play request that arrived before the window listened, once; nil when there is none.
func (s *Service) Take() *Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.pending
	s.pending = nil
	return r
}

// Create makes a shortcut named after the profile and game that plays it, and returns where it was written.
func (s *Service) Create(game, gameName, profile, profileName string) (string, error) {
	if !validID(game) || !validID(profile) {
		return "", errors.New("a shortcut needs a game and a profile")
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return create(nxm.Launchable(exe), Arg(game, profile), profileName+" ("+gameName+")")
}
