// Package picker brings archives into the window: the open-file dialog and the drop event.
package picker

import "github.com/wailsapp/wails/v3/pkg/application"

// DroppedEvent is emitted with the paths of the files dropped on the window.
const DroppedEvent = "files:dropped"

// Service exposes the file dialog to the frontend.
type Service struct {
	// App is set after application.New.
	App *application.App
}

// PickArchives asks for archives and returns the chosen paths, none when the dialog is cancelled.
func (s *Service) PickArchives() ([]string, error) {
	d := s.App.Dialog.OpenFile().
		SetTitle("Add archive").
		AddFilter("Archives (zip, RAR, 7z)", "*.zip;*.rar;*.7z").
		AddFilter("All files", "*")
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	return d.PromptForMultipleSelection()
}
