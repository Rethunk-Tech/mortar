// Package shortcut makes desktop shortcuts that play one profile, and takes the play request such a shortcut sends.
// A shortcut runs Mortar with --play=<game>/<profile>; a running Mortar receives it as a second instance, and a
// closed one starts with it, so the window plays the profile through its usual Play path either way.
package shortcut

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/sandbox"
	"github.com/Rethunk-AI/mortar/internal/selfexe"
	"github.com/Rethunk-AI/mortar/internal/steam"
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
	Covers  func(game, profile string) ([]string, error)
	mu      sync.Mutex
	pending *Request
}

// Receive keeps a play request found in args and tells the window; it reports whether there was one.
//
//wails:ignore
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

// Capabilities says which shortcut actions this install can do, so the UI can hide the rest.
type Capabilities struct {
	Flatpak    bool `json:"flatpak"`
	Shortcuts  bool `json:"shortcuts"`
	AddToSteam bool `json:"addToSteam"`
}

// Capabilities reports what shortcut actions are available; Add to Steam is unavailable inside a Flatpak.
func (s *Service) Capabilities() Capabilities {
	flatpak := sandbox.InFlatpak()
	return Capabilities{Flatpak: flatpak, Shortcuts: true, AddToSteam: !flatpak}
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
	return create(selfexe.Launchable(exe), Arg(game, profile), profileName+" ("+gameName+")")
}

// Exists reports whether a desktop or Start-menu shortcut plays the profile.
func (s *Service) Exists(game, profile string) (bool, error) {
	return exists(game, profile)
}

// Remove removes the desktop or Start-menu shortcut for the profile.
func (s *Service) Remove(game, profile string) error {
	return Removed(game, profile)
}

// ErrFlatpakSteamShortcut means the profile's shortcut cannot go into Steam from a Flatpak: Steam on the host
// cannot start Mortar's /app/bin/mortar, and the sandbox cannot edit Steam's files without broad host access.
var ErrFlatpakSteamShortcut = errors.New("adding to Steam is not available in the Flatpak: use the desktop shortcut, or add Mortar to Steam as a non-Steam game with the command `flatpak run tech.rethunk.Mortar --play=<game>/<profile>`")

// ErrSteamRunning means Steam is open; it rewrites its shortcut list when it exits, which would drop the new entry.
var ErrSteamRunning = errors.New("close Steam first: it rewrites its game list when it exits")

func firstLocalFile(paths []string) string {
	for _, path := range paths {
		if path == "" || strings.Contains(path, "://") {
			continue
		}
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

// AddToSteam adds a non-Steam game that plays the profile to the native Steam library, for Big Picture and the
// Steam Deck's Game Mode. It reports false when the same shortcut is already there.
func (s *Service) AddToSteam(game, gameName, profile, profileName string) (bool, error) {
	if !validID(game) || !validID(profile) {
		return false, errors.New("a shortcut needs a game and a profile")
	}
	if sandbox.InFlatpak() {
		return false, ErrFlatpakSteamShortcut
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	st, status := steam.Locate(home)
	if status != steam.Found || st.Kind != steam.KindNative {
		return false, errors.New("no Steam was found that can start Mortar: Steam (Flatpak) cannot run programs outside its sandbox")
	}
	if running, err := launch.Processes("/proc", "steam"); err == nil && len(running) > 0 {
		return false, ErrSteamRunning
	}
	exe, err := os.Executable()
	if err != nil {
		return false, err
	}
	exe = selfexe.Launchable(exe)
	cover := ""
	if s.Covers != nil {
		if paths, err := s.Covers(game, profile); err == nil {
			cover = firstLocalFile(paths)
		}
	}
	return st.AddShortcut(steam.Shortcut{
		Name:          profileName + " (" + gameName + ")",
		Exe:           exe,
		StartDir:      filepath.Dir(exe),
		LaunchOptions: Arg(game, profile),
		Cover:         cover,
	})
}
