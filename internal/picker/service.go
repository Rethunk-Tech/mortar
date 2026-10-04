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
	d.AttachToWindow(s.App.Window.Current())
	return d.PromptForMultipleSelection()
}

// PickImage asks for one image and returns its path, or "" when the dialog is cancelled.
func (s *Service) PickImage(title string) (string, error) {
	d := s.App.Dialog.OpenFile().
		SetTitle(title).
		AddFilter("Images (PNG, JPEG, WebP)", "*.png;*.jpg;*.jpeg;*.webp")
	d.AttachToWindow(s.App.Window.Current())
	return d.PromptForSingleSelection()
}

// SaveFile asks where to write contents and writes them. It returns "" when the dialog is cancelled.
func (s *Service) SaveFile(title, filename, contents string) (string, error) {
	return SaveFile(s.App, title, filename, "Text files", "*.txt", []byte(contents))
}

// SaveFile asks where to write data through the native save dialog, offering one filter plus all files,
// and writes it owner-only. It returns "" when the dialog is cancelled.
func SaveFile(app *application.App, title, filename, filterName, pattern string, data []byte) (string, error) {
	d := app.Dialog.SaveFile()
	d.SetOptions(&application.SaveFileDialogOptions{Title: title, Filename: filename})
	d.AddFilter(filterName, pattern)
	d.AddFilter("All files", "*")
	d.AttachToWindow(app.Window.Current())
	path, err := d.PromptForSingleSelection()
	if err != nil || path == "" {
		return path, err
	}
	return path, fsx.WriteFile(path, data, 0o600)
}
