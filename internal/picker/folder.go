package picker

import "github.com/Rethunk-Tech/mortar/internal/winhost"

// PickFolder asks for a folder and returns its path, or "" when the dialog is cancelled.
func (s *Service) PickFolder(title string) (string, error) {
	return s.App.OpenFile(winhost.Dialog{Title: title, Folder: true})
}
