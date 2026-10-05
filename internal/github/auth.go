package github

import (
	"context"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	tokenTimeout = 3 * time.Second
	tokenTTL     = 5 * time.Minute
)

// Auth lends requests the token of the user's `gh` login. Without gh, or when gh is logged out or fails, requests
// go unauthenticated. The token is never logged or put into an error.
type Auth struct {
	// Run prints the token; nil runs `gh auth token`.
	Run func(ctx context.Context) ([]byte, error)

	mu      sync.Mutex
	token   string
	fetched time.Time
}

// DefaultAuth authorizes every request this package makes to GitHub.
var DefaultAuth = &Auth{}

func ghAuthToken(ctx context.Context) ([]byte, error) {
	return exec.CommandContext(ctx, "gh", "auth", "token").Output() // #nosec G204 -- fixed arguments
}

func (a *Auth) get(ctx context.Context) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.fetched.IsZero() && time.Since(a.fetched) < tokenTTL {
		return a.token
	}
	run := a.Run
	if run == nil {
		run = ghAuthToken
	}
	ctx, cancel := context.WithTimeout(ctx, tokenTimeout)
	defer cancel()
	out, err := run(ctx)
	a.token, a.fetched = "", time.Now()
	if err == nil {
		a.token = strings.TrimSpace(string(out))
	}
	return a.token
}

// IsGitHubHost reports whether host is one that the gh token may be sent to.
func IsGitHubHost(host string) bool {
	return host == "github.com" || host == "api.github.com"
}

// Apply sets the bearer token on req when it targets GitHub itself and a token is available. A redirect to another
// host (release asset storage) drops the header in net/http.
func (a *Auth) Apply(req *http.Request) {
	if !IsGitHubHost(req.URL.Hostname()) {
		return
	}
	if t := a.get(req.Context()); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
}
