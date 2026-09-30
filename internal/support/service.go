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

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/problems"
)

const (
	// MaxLog allows the console's launch.MaxLines at 256 bytes a line; a SMAPI log past that is not one session's.
	MaxLog = launch.MaxLines * 256

	smapiBase = "https://smapi.io"
	issuesURL = "https://github.com/Rethunk-AI/mortar/issues/new"
	timeout   = 60 * time.Second
)

// Service reads and uploads the log, and builds the bug report link.
type Service struct {
	base    string
	version string
	env     func(gameID string) problems.Environment
	client  *http.Client
}

// NewService takes Mortar's version and the environment reader the bug report quotes.
func NewService(version string, env func(gameID string) problems.Environment) *Service {
	return newService(smapiBase, version, env)
}

func newService(base, version string, env func(string) problems.Environment) *Service {
	return &Service{base: base, version: version, env: env, client: &http.Client{
		Timeout: timeout,
		// smapi.io answers success with a redirect to the log's page and failure with a 200 page, so the redirect
		// itself is the answer.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// Log returns the game's loader log as written on disk, or "" when there is none yet.
func (s *Service) Log(gameID string) (string, error) {
	g := game.Find(gameID)
	if g == nil {
		return "", fmt.Errorf("unknown game %q", gameID)
	}
	path, err := g.LogFile()
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
	return strings.ToValidUTF8(string(data), ""), nil
}

// Upload posts the log to smapi.io and returns its page's link.
func (s *Service) Upload(ctx context.Context, log string) (string, error) {
	if strings.TrimSpace(log) == "" {
		return "", errors.New("the log is empty")
	}
	if len(log) > MaxLog {
		return "", errors.New("the log is too large to upload")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/log",
		strings.NewReader(url.Values{"input": {log}}.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 300 || resp.StatusCode > 399 {
		if resp.StatusCode == http.StatusOK {
			return "", errors.New("smapi.io did not accept the log")
		}
		return "", fmt.Errorf("smapi.io answered %s", resp.Status)
	}
	loc, err := resp.Location()
	if err != nil {
		return "", err
	}
	id, ok := strings.CutPrefix(loc.Path, "/log/")
	if !ok || id == "" || strings.Contains(id, "/") {
		return "", fmt.Errorf("smapi.io redirected to %q, not a log page", loc.Path)
	}
	return s.base + "/log/" + url.PathEscape(id), nil
}

// BugURL is a new GitHub issue on Mortar prefilled with the version, OS and game, and no log.
func (s *Service) BugURL(gameID string) string {
	about := fmt.Sprintf("Mortar %s · %s/%s", s.version, runtime.GOOS, runtime.GOARCH)
	if g := game.Find(gameID); g != nil {
		env := s.env(gameID)
		about += " · " + g.Name()
		if env.GameVersion != "" {
			about += " " + env.GameVersion
		}
		if env.APIVersion != "" {
			about += fmt.Sprintf(" (%s %s)", g.LoaderName(), env.APIVersion)
		}
	}
	body := "**What happened**\n\n\n**What you expected**\n\n\n**Steps to reproduce**\n\n\n---\n" + about + "\n"
	return issuesURL + "?" + url.Values{"title": {"Bug: "}, "body": {body}}.Encode()
}
