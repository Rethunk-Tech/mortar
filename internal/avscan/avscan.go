// Package avscan asks the player's antivirus about extracted mod files before they enter the store: Windows' AMSI,
// a clamd daemon, or a command the player names.
package avscan

import (
	"context"
	"runtime"

	goavscan "github.com/Rethunk-Tech/go-avscan"
)

// Modes of Config, as the antivirus setting names them.
const (
	ModeAutomatic = "automatic"
	ModeClamd     = "clamd"
	ModeCommand   = "command"
	ModeOff       = "off"
)

// Config is the antivirus settings: which scanner, clamd's socket and the custom command.
type Config struct {
	Mode    string
	Socket  string
	Command string
}

// Chosen is the scanner the settings name: Windows' AMSI, a clamd daemon, the custom command, or none.
type Chosen struct{ inner goavscan.Scanner }

// New is the scanner the settings name; the empty mode is Automatic.
func New(c Config) Chosen {
	switch c.Mode {
	case ModeOff:
		return Chosen{off{}}
	case ModeClamd:
		return Chosen{goavscan.Clamd{Socket: c.Socket}}
	case ModeCommand:
		return Chosen{goavscan.Command{Line: c.Command}}
	default:
		if runtime.GOOS == "windows" {
			return Chosen{goavscan.AMSI{AppName: "Mortar"}}
		}
		return Chosen{goavscan.Clamd{Socket: c.Socket}}
	}
}

// Scan runs the chosen scanner. AMSI and a command that prints nothing flag a file without naming the threat, so the
// name is worded here.
func (c Chosen) Scan(ctx context.Context, dir string) (goavscan.Detection, bool, error) {
	d, found, err := c.inner.Scan(ctx, dir)
	if found && d.Name == "" {
		switch c.inner.(type) {
		case goavscan.AMSI:
			d.Name = "malware reported by the Windows antivirus"
		case goavscan.Command:
			d.Name = "detected by " + d.Scanner
		}
	}
	return d, found, err
}

type off struct{}

func (off) Scan(context.Context, string) (goavscan.Detection, bool, error) {
	return goavscan.Detection{}, false, nil
}
