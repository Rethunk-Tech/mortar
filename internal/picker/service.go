// Package picker brings files into the window: the open-file dialogs and the drop event.
package picker

import (
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/wailsapp/wails/v3/pkg/application"
)

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

// PickImage asks for one image and returns its path, or "" when the dialog is cancelled.
func (s *Service) PickImage(title string) (string, error) {
	d := s.App.Dialog.OpenFile().
		SetTitle(title).
		AddFilter("Images (PNG, JPEG, WebP)", "*.png;*.jpg;*.jpeg;*.webp")
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	return d.PromptForSingleSelection()
}

// SaveFile asks where to write contents and writes them. It returns "" when the dialog is cancelled.
func (s *Service) SaveFile(title, filename, contents string) (string, error) {
	d := s.App.Dialog.SaveFile()
	d.SetOptions(&application.SaveFileDialogOptions{Title: title, Filename: filename})
	d.AddFilter("Text files", "*.txt")
	d.AddFilter("All files", "*")
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	path, err := d.PromptForSingleSelection()
	if err != nil || path == "" {
		return path, err
	}
	return path, fsx.WriteFile(path, []byte(contents), 0o600)
}
