// Package picker brings files into the window: the open-file dialogs and the drop event.
package picker

import (
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/winhost"
)

// DroppedEvent is emitted with the paths of the files dropped on the window.
const DroppedEvent = "files:dropped"

// Service exposes the file dialog to the frontend.
type Service struct {
	// App is set after application.New.
	App winhost.Host
}

// PickArchives asks for archives and returns the chosen paths, none when the dialog is cancelled.
func (s *Service) PickArchives() ([]string, error) {
	return s.App.OpenFiles(winhost.Dialog{Title: "Add archive", Filters: []winhost.Filter{
		{Name: "Archives (zip, RAR, 7z, tar, gz, xz, zst, bz2, lzma)", Pattern: archive.PickerPattern()},
		{Name: "All files", Pattern: "*"},
	}})
}

// PickImage asks for one image and returns its path, or "" when the dialog is cancelled.
func (s *Service) PickImage(title string) (string, error) {
	return s.App.OpenFile(winhost.Dialog{Title: title, Filters: []winhost.Filter{{Name: "Images (PNG, JPEG, WebP)", Pattern: "*.png;*.jpg;*.jpeg;*.webp"}}})
}

// PickPackFile asks for an r2modman profile export (.r2z) or a Thunderstore modpack zip and returns its path, or "" when
// the dialog is cancelled.
func (s *Service) PickPackFile() (string, error) {
	return s.App.OpenFile(winhost.Dialog{Title: "Open a profile export", Filters: []winhost.Filter{
		{Name: "r2modman or Gale export (.r2z, .zip)", Pattern: "*.r2z;*.zip"},
		{Name: "All files", Pattern: "*"},
	}})
}

// SaveFile asks where to write contents and writes them. It returns "" when the dialog is cancelled.
func (s *Service) SaveFile(title, filename, contents string) (string, error) {
	return SaveFile(s.App, title, filename, "Text files", "*.txt", []byte(contents))
}

// SaveFile asks where to write data through the native save dialog, offering one filter plus all files,
// and writes it owner-only. It returns "" when the dialog is cancelled.
func SaveFile(app winhost.Host, title, filename, filterName, pattern string, data []byte) (string, error) {
	path, err := app.SaveFile(winhost.Dialog{Title: title, Filename: filename, Filters: []winhost.Filter{{Name: filterName, Pattern: pattern}, {Name: "All files", Pattern: "*"}}})
	if err != nil || path == "" {
		return path, err
	}
	return path, fsx.WriteFile(path, data, 0o600)
}
