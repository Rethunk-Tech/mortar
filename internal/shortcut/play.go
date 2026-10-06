package shortcut

// WindowMode is how play mode shows Mortar's window.
type WindowMode string

const (
	// WindowPrompt is a small window that holds only the dialog saying why Play is blocked.
	WindowPrompt WindowMode = "prompt"
	// WindowHidden hides the window while the game starts and runs.
	WindowHidden WindowMode = "hidden"
	// WindowFull is Mortar's usual window, once the user chose to fix what blocked Play.
	WindowFull WindowMode = "full"
)

// StartSolo takes the play request Mortar was started with as play mode, for a shortcut run from Steam's Game Mode:
// the window stays hidden unless Play is blocked, and Mortar exits when the game closes, so Steam counts the
// shortcut's playtime. It reports whether there was a request.
//
//wails:ignore
func (s *Service) StartSolo(args []string) bool {
	r, ok := Parse(args)
	if !ok {
		return false
	}
	s.mu.Lock()
	s.solo = &r
	s.mu.Unlock()
	return s.Receive(args)
}

// PlayMode reports whether Mortar runs only to play the profile it was started with.
func (s *Service) PlayMode() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.solo != nil
}

// ShowPrompt shows the small window that asks what to do about a blocked Play.
func (s *Service) ShowPrompt() {
	if s.PlayMode() {
		s.window(WindowPrompt)
	}
}

// HidePrompt hides the window again once the game is starting.
func (s *Service) HidePrompt() {
	if s.PlayMode() {
		s.window(WindowHidden)
	}
}

// LeavePlayMode turns play mode into a normal session in the full window, which stays open after the game closes.
func (s *Service) LeavePlayMode() {
	s.mu.Lock()
	s.solo = nil
	s.mu.Unlock()
	s.window(WindowFull)
}

func (s *Service) window(m WindowMode) {
	if s.Window != nil {
		s.Window(m)
	}
}

// Observe follows the launch state of game and reports when play mode should exit: the played game ran and is idle
// again. Other games, and a game that never reached running, never end it.
//
//wails:ignore
func (s *Service) Observe(game string, running bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.solo == nil || s.solo.Game != game {
		return false
	}
	if running {
		s.ran = true
		return false
	}
	return s.ran
}
