package launchsvc

import (
	"os"
	"path/filepath"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// earlyExitWindow is how soon after Play a game must exit to count as having failed at startup. Launch tracking dates
// a run from the moment Mortar starts it, before the process exists, and bisect already allows a healthy game 20s to
// reach its title screen (bisectStartupGrace) and 10s more to hold a scene (bisectSceneSettle). Unity creates its
// graphics device before the first scene loads, so a game still alive after 30s got past that step.
//
// The exit is judged on its timing alone: no graphics-device failure line from a real public Player.log could be
// verified (the r2modman issue about PEAK and Vulkan, ebkr/r2modmanPlus#1842, quotes none), and a line taken from
// memory would misclassify exits.
const earlyExitWindow = 30 * time.Second

// maxCrashAsks is how many times one run of early exits may bring the question back: the player is asked after the
// first, and once more if they keep the same choice and it fails again, then left alone.
const maxCrashAsks = 2

const graphicsCrashFile = "graphics-crash.json"

// graphicsMarker is a profile's record of games that exited at startup under a graphics choice other than the
// recommended one. Pending asks on the next Play; Asked counts the questions that run of failures has already cost.
type graphicsMarker struct {
	Pending bool `json:"pending"`
	Asked   int  `json:"asked"`
}

// afterRun is the marker once a run ended after d: an early exit asks again while questions remain, and a run that
// lasted clears the record.
func (m graphicsMarker) afterRun(d time.Duration) graphicsMarker {
	if d >= earlyExitWindow {
		return graphicsMarker{}
	}
	m.Pending = m.Asked < maxCrashAsks
	return m
}

// afterAnswer is the marker once the player answered: choosing the recommendation ends the record, any other choice
// spends the pending question.
func (m graphicsMarker) afterAnswer(recommended bool) graphicsMarker {
	if recommended {
		return graphicsMarker{}
	}
	if m.Pending {
		m.Asked++
		m.Pending = false
	}
	return m
}

func (s *Service) graphicsMarkerPath(gameID, profileID string) (string, error) {
	dir, err := s.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, startupDir, graphicsCrashFile), nil
}

func (s *Service) readGraphicsMarker(gameID, profileID string) (graphicsMarker, error) {
	var m graphicsMarker
	path, err := s.graphicsMarkerPath(gameID, profileID)
	if err != nil {
		return m, err
	}
	_, err = datadir.ReadJSON(path, &m)
	return m, err
}

func (s *Service) writeGraphicsMarker(gameID, profileID string, m graphicsMarker) error {
	path, err := s.graphicsMarkerPath(gameID, profileID)
	if err != nil {
		return err
	}
	if m == (graphicsMarker{}) {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(path, m)
}

// noteGraphicsExit updates the profile's marker for a run that ended after d. A choice that is the catalog's
// recommendation is never blamed: only an unset or other choice can be what the player should be steered off.
func (s *Service) noteGraphicsExit(g game.Game, profileID string, d time.Duration) {
	offer := graphicsOffer(g.ID())
	if offer == nil || s.settings == nil || s.profiles == nil {
		return
	}
	chosen, _ := chosenGraphics(offer, s.settings.Get(), settings.Scope{Game: g.ID(), Install: installOf(g), Profile: profileID}, launchOverrides(s.profiles, g.ID(), profileID))
	m, err := s.readGraphicsMarker(g.ID(), profileID)
	if err != nil {
		return
	}
	next := graphicsMarker{}
	if chosen.ID != offer.Recommended {
		next = m.afterRun(d)
	}
	if next != m {
		_ = s.writeGraphicsMarker(g.ID(), profileID, next)
	}
}

// GraphicsAnswered records that the player answered Play's graphics question with choice.
func (s *Service) GraphicsAnswered(gameID, profileID, choice string) error {
	offer := graphicsOffer(gameID)
	if offer == nil {
		return nil
	}
	m, err := s.readGraphicsMarker(gameID, profileID)
	if err != nil {
		return err
	}
	return s.writeGraphicsMarker(gameID, profileID, m.afterAnswer(choice == offer.Recommended))
}

// noteStartedProcessExit is noteGraphicsExit for a launch whose started process exited before the game counted as
// running, which is how a direct start that dies at once ends: the run is recorded as failed, but the game did start.
func (s *Service) noteStartedProcessExit(g game.Game, profileID string, buf *launch.Buffer) {
	s.mu.Lock()
	sess := s.logs[keyOf(g)]
	s.mu.Unlock()
	if sess.buf != buf || sess.vanilla || profileID == "" {
		return
	}
	s.noteGraphicsExit(g, profileID, time.Since(sess.started))
}
