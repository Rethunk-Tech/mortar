package avscan

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	goavscan "github.com/Rethunk-Tech/go-avscan"
	"github.com/Rethunk-Tech/go-safefs"
)

const productTimeout = 5 * time.Second

// ProductLine is the scanner line Settings shows: Text in English for the CLI, and for a line Mortar worded itself a
// Code (and Params) the window words in the user's language. A product's own name has no Code.
type ProductLine struct {
	Text   string            `json:"text"`
	Code   string            `json:"code,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

// Product names the scanner the settings resolve to, for Settings: the antivirus Windows Security Center lists, the
// clamd version found, the custom command's program, or what is missing.
func Product(ctx context.Context, c Config) string { return ProductOf(ctx, c).Text }

// ProductOf is Product with the code the window words it from.
func ProductOf(ctx context.Context, c Config) ProductLine {
	ctx, cancel := context.WithTimeout(ctx, productTimeout)
	defer cancel()
	switch c.Mode {
	case ModeOff:
		return ProductLine{Text: "Antivirus scanning is off", Code: "off"}
	case ModeCommand:
		if prog := (goavscan.Command{Line: c.Command}).Program(); prog != "" {
			return ProductLine{Text: prog}
		}
		return ProductLine{Text: "No command set", Code: "noCommand"}
	case ModeClamd:
		return clamdProduct(ctx, goavscan.Clamd{Socket: c.Socket})
	}
	if runtime.GOOS == "windows" {
		return windowsProduct(ctx)
	}
	return clamdProduct(ctx, goavscan.Clamd{Socket: c.Socket})
}

func clamdProduct(ctx context.Context, c goavscan.Clamd) ProductLine {
	line, err := c.Version(ctx)
	if err != nil || line == "" {
		return ProductLine{Text: "No antivirus found", Code: "none"}
	}
	version, _, _ := strings.Cut(line, "/")
	return ProductLine{Text: strings.Replace(version, "ClamAV", "clamd", 1)}
}

// windowsProduct reads the product names from Security Center; without a readable answer AMSI still reaches whichever
// antivirus is registered, so that is what it says.
func windowsProduct(ctx context.Context) ProductLine {
	fallback := ProductLine{Text: "Windows antivirus (AMSI)", Code: "windowsAmsi"}
	names, err := goavscan.SecurityCenterProducts(ctx)
	if err != nil {
		return fallback
	}
	if name := strings.Join(names, ", "); name != "" {
		return ProductLine{Text: fmt.Sprintf("%s (AMSI)", name), Code: "namedAmsi", Params: map[string]string{"names": name}}
	}
	return fallback
}

// Status is what Settings and `mortar antivirus status` report: which scanner the settings resolve to, the product
// behind it, and whether a scan can run now.
type Status struct {
	Mode    string `json:"mode"`
	Scanner string `json:"scanner"`
	Product string `json:"product"`
	Ready   bool   `json:"ready"`
	Problem string `json:"problem,omitempty"`
}

// StatusOf resolves c. Ready comes from scanning an empty folder, which opens the scanner (an AMSI session, a clamd
// connection, the command's program) without any file to flag.
func StatusOf(ctx context.Context, c Config) Status {
	st := Status{Mode: c.Mode, Scanner: kindOf(c), Product: Product(ctx, c)}
	if st.Mode == "" {
		st.Mode = ModeAutomatic
	}
	dir, err := os.MkdirTemp("", "mortar-av-status-")
	if err != nil {
		st.Problem = err.Error()
		return st
	}
	defer func() { _ = safefs.RemoveAll(dir) }()
	ctx, cancel := context.WithTimeout(ctx, productTimeout)
	defer cancel()
	_, _, err = New(c).Scan(ctx, dir)
	if err == nil && st.Scanner == "clamd" {
		// An empty folder never reaches the daemon, so ask it directly.
		_, err = goavscan.Clamd{Socket: c.Socket}.Version(ctx)
	}
	if err != nil {
		st.Problem = err.Error()
		return st
	}
	st.Ready = true
	return st
}

func kindOf(c Config) string {
	switch c.Mode {
	case ModeOff:
		return "off"
	case ModeClamd:
		return "clamd"
	case ModeCommand:
		return "command"
	}
	if runtime.GOOS == "windows" {
		return "amsi"
	}
	return "clamd"
}
