package picker

import "github.com/Rethunk-Tech/mortar/internal/winhost"

// PickExecutable asks for one program and returns its path, or "" when the dialog is cancelled.
func (s *Service) PickExecutable(title string) (string, error) {
	return s.App.OpenFile(winhost.Dialog{Title: title})
}
