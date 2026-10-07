// Package avscan asks the player's antivirus about extracted mod files before they enter the store: Windows' AMSI,
// a clamd daemon, or a command the player names.
package avscan

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"runtime"
)

// Modes of Config, as the antivirus setting names them.
const (
	ModeAutomatic = "automatic"
	ModeClamd     = "clamd"
	ModeCommand   = "command"
	ModeOff       = "off"
)

// Modes lists the values of the antivirus setting.
var Modes = []string{ModeAutomatic, ModeClamd, ModeCommand, ModeOff}

// ErrNoScanner is a scanner that is not there: no AMSI provider, no reachable clamd. It is not a failure to report,
// since most Linux players run no daemon.
var ErrNoScanner = errors.New("no antivirus scanner available")

// Detection is what a scanner flagged: its name for the threat and the file, relative to the scanned folder, when the
// scanner says which.
type Detection struct {
	Name    string `json:"name"`
	File    string `json:"file"`
	Scanner string `json:"scanner"`
}

// Scanner scans a folder of extracted files. The bool says whether anything was detected; an error is a scan that did
// not finish and says nothing about the files.
type Scanner interface {
	Scan(ctx context.Context, dir string) (Detection, bool, error)
}

// Config is the antivirus settings: which scanner, clamd's socket and the custom command.
type Config struct {
	Mode    string
	Socket  string
	Command string
}

// Chosen is the scanner the settings name: Windows' AMSI, a clamd daemon, the custom command, or none.
type Chosen struct{ inner Scanner }

// New is the scanner the settings name; the empty mode is Automatic.
func New(c Config) Chosen {
	switch c.Mode {
	case ModeOff:
		return Chosen{off{}}
	case ModeClamd:
		return Chosen{clamd{socket: c.Socket}}
	case ModeCommand:
		return Chosen{command{line: c.Command}}
	default:
		if runtime.GOOS == "windows" {
			return Chosen{amsi{}}
		}
		return Chosen{clamd{socket: c.Socket}}
	}
}

// Scan runs the chosen scanner.
func (c Chosen) Scan(ctx context.Context, dir string) (Detection, bool, error) {
	return c.inner.Scan(ctx, dir)
}

type off struct{}

func (off) Scan(context.Context, string) (Detection, bool, error) { return Detection{}, false, nil }

// files calls visit for each regular file under dir with its path relative to dir, stopping at the first error or
// when the context ends.
func files(ctx context.Context, dir string, visit func(abs, rel string) error) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if cerr := ctx.Err(); cerr != nil {
			return cerr
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		return visit(path, filepath.ToSlash(rel))
	})
}
