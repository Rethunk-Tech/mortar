// Package support uploads the game's loader log to smapi.io and builds the link for reporting a Mortar bug.
package support

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/doctor"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/logshare"
	"github.com/Rethunk-Tech/mortar/internal/nativehost"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	// MaxLog allows the console's launch.MaxLines at 256 bytes a line; a SMAPI log past that is not one session's.
	MaxLog = launch.MaxLines * 256

	smapiBase = "https://smapi.io"
	issuesURL = "https://github.com/Rethunk-Tech/mortar/issues/new"
	timeout   = 60 * time.Second
)

// Service reads and uploads the log, and builds the bug report link.
type Service struct {
	base    string
	version string
	env     func(gameID string) problems.Environment
	home    string
	modsDir func(gameID, profileID string) (string, error)
	client  *http.Client

	// Dir is the user data folder. Nil uses datadir.Dir. Tests point it at a temp fixture.
	Dir func() (string, error)
	// SaveZip writes the diagnostics zip after the user picks a path. Nil uses App's save dialog.
	SaveZip func(title, filename string, data []byte) (string, error)
	// RecentLog is in-memory log lines when Mortar has no log file. Nil means none.
	RecentLog func(gameID, profileID string) string
	// App is set after application.New so SaveDiagnostics can show the save dialog.
	App *application.App
}

// ExtensionContact is the last time a browser extension talked to Mortar's native host; empty before any did.
func (s *Service) ExtensionContact() nativehost.Contact {
	return nativehost.LastContact()
}

// NewService takes Mortar's version, the environment reader the bug report quotes, and the user's home and profile
// mods folders that tell whose the log is.
func NewService(version string, env func(gameID string) problems.Environment, home string, modsDir func(gameID, profileID string) (string, error)) *Service {
	s := newService(smapiBase, version, env)
	s.home, s.modsDir = home, modsDir
	return s
}

func newService(base, version string, env func(string) problems.Environment) *Service {
	return &Service{base: base, version: version, env: env, client: &http.Client{
		Timeout: timeout,
		// smapi.io answers success with a redirect to the log's page and failure with a 200 page, so the redirect
		// itself is the answer.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// Log returns the game's loader log as written on disk, or "" when there is none yet or it is another profile's.
func (s *Service) Log(gameID, profileID string) (string, error) {
	if _, err := game.Require(gameID); err != nil {
		return "", err
	}
	path, err := game.LogFile(gameID)
	if err != nil {
		return "", err
	}
	data, err := fsx.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if len(data) > MaxLog {
		return "", fmt.Errorf("the log is %d MB, too large to upload", len(data)>>20)
	}
	modsDir, err := s.modsDir(gameID, profileID)
	if err != nil {
		return "", err
	}
	text := strings.ToValidUTF8(string(data), "")
	if !launch.LogOwnedBy(text, s.home, modsDir) {
		return "", nil
	}
	return text, nil
}

// Upload posts the log to smapi.io and returns its page's link.
func (s *Service) Upload(ctx context.Context, log string) (string, error) {
	if strings.TrimSpace(log) == "" {
		return "", errors.New("the log is empty")
	}
	if len(log) > MaxLog {
		return "", errors.New("the log is too large to upload")
	}
	c := &logshare.Client{BaseURL: strings.TrimRight(s.base, "/") + "/log", HTTP: s.client}
	return c.Upload(ctx, log)
}

// BugReport is what the user wrote in Mortar's Report a bug dialog.
type BugReport struct {
	Title       string `json:"title"`
	Happened    string `json:"happened"`
	Expected    string `json:"expected"`
	Steps       string `json:"steps"`
	Diagnostics bool   `json:"diagnostics"`
}

// BugURL is a new GitHub issue on Mortar prefilled with the report, the version, OS and game, and, when the user
// left it on, the diagnostics checks and recent log lines; never a full log.
func (s *Service) BugURL(gameID string, r BugReport) string {
	about := fmt.Sprintf("Mortar %s · %s/%s", s.version, runtime.GOOS, runtime.GOARCH)
	if g := game.Find(gameID); g != nil {
		env := s.env(gameID)
		about += " · " + g.Name()
		if env.GameVersion != "" {
			about += " " + env.GameVersion
		}
		if env.APIVersion != "" {
			about += fmt.Sprintf(" (%s %s)", game.LoaderName(gameID, ""), env.APIVersion)
		}
	}
	section := func(heading, text string) string {
		return "**" + heading + "**\n" + strings.TrimSpace(text) + "\n\n"
	}
	body := section("What happened", r.Happened) + section("What you expected", r.Expected) +
		section("Steps to reproduce", r.Steps) + "---\n" + about + "\n"
	if r.Diagnostics {
		if report, err := s.Doctor(); err == nil {
			dir, _ := s.dataDir()
			body += diagnosticsSection(report, s.home, dir)
		}
	}
	title := strings.TrimSpace(r.Title)
	if title == "" {
		title = "Bug: "
	}
	return issuesURL + "?" + url.Values{"title": {title}, "body": {body}}.Encode()
}

// maxDiagnostics keeps the prefilled issue URL well under the length GitHub and browsers accept.
const maxDiagnostics = 3000

// diagnosticsSection lists each diagnostics check as "status: detail" with the home folder shown as ~,
// cut short once it would make the issue link too long.
func diagnosticsSection(report doctor.Report, home, dataDir string) string {
	var b strings.Builder
	b.WriteString("\n**Diagnostics**\n```\n")
	for _, c := range report.Checks {
		line := c.Status + ": " + hideHome(c.Detail, home) + "\n"
		if b.Len()+len(line) > maxDiagnostics {
			b.WriteString("…\n")
			break
		}
		b.WriteString(line)
	}
	b.WriteString("```\n")
	b.WriteString(logTailSections(dataDir, home, maxDiagnostics-b.Len()))
	return b.String()
}
