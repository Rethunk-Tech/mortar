// Package picker brings files into the window: the open-file dialogs and the drop event.
package picker

import (
	"errors"
	"os"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/i18n"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
	return s.App.OpenFiles(winhost.Dialog{Title: i18n.T("Add archive"), Filters: []winhost.Filter{
		{Name: i18n.T("Archives (zip, RAR, 7z, tar, gz, xz, zst, bz2, lz4, lzma)"), Pattern: archive.PickerPattern()},
		{Name: i18n.T("All files"), Pattern: "*"},
	}})
}

// PickImage asks for one image and returns its path, or "" when the dialog is cancelled.
func (s *Service) PickImage(title string) (string, error) {
	return s.App.OpenFile(winhost.Dialog{Title: title, Filters: []winhost.Filter{{Name: i18n.T("Images (PNG, JPEG, WebP)"), Pattern: archive.CaseGlobs("*.png;*.jpg;*.jpeg;*.webp")}}})
}

// PickPackFile asks for an r2modman profile export (.r2z) or a Thunderstore modpack zip and returns its path, or "" when
// the dialog is cancelled.
func (s *Service) PickPackFile() (string, error) {
	return s.App.OpenFile(winhost.Dialog{Title: i18n.T("Open a profile export"), Filters: []winhost.Filter{
		{Name: i18n.T("{app} export (.r2z, .zip)", "app", "r2modman or Gale"), Pattern: archive.CaseGlobs("*.r2z;*.zip")},
		{Name: i18n.T("All files"), Pattern: "*"},
	}})
}

// maxTextBytes bounds a text file read into the window.
const maxTextBytes = 1 << 20

// ReadTextFile asks for a text file and returns what it holds, or "" when the dialog is cancelled.
func (s *Service) ReadTextFile(title string) (string, error) {
	path, err := s.App.OpenFile(winhost.Dialog{Title: title, Filters: []winhost.Filter{
		{Name: i18n.T("Text files"), Pattern: archive.CaseGlobs("*.txt;*.md;*.json")},
		{Name: i18n.T("All files"), Pattern: "*"},
	}})
	if err != nil || path == "" {
		return "", err
	}
	f, err := os.Open(path) //nolint:gosec // G304: the user picked this path; it follows a picked symlink, which fsx.Open refuses
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	text, err := fsx.ReadCapped(f, maxTextBytes)
	if errors.Is(err, fsx.ErrTooLarge) {
		return "", usererr.New(usererr.Invalid, "that file is too large")
	}
	return string(text), err
}

// SaveFile asks where to write contents and writes them. It returns "" when the dialog is cancelled.
func (s *Service) SaveFile(title, filename, contents string) (string, error) {
	return SaveFile(s.App, title, filename, i18n.T("Text files"), "*.txt", []byte(contents))
}

// SaveFile asks where to write data through the native save dialog, offering one filter plus all files,
// and writes it owner-only. It returns "" when the dialog is cancelled.
func SaveFile(app winhost.Host, title, filename, filterName, pattern string, data []byte) (string, error) {
	path, err := app.SaveFile(winhost.Dialog{Title: title, Filename: filename, Filters: []winhost.Filter{{Name: filterName, Pattern: pattern}, {Name: i18n.T("All files"), Pattern: "*"}}})
	if err != nil || path == "" {
		return path, err
	}
	return path, fsx.WriteFile(path, data, 0o600)
}
