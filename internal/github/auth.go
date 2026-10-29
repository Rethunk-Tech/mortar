package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/nowindow"
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
	cmd := exec.CommandContext(ctx, "gh", "auth", "token") // #nosec G204 -- fixed arguments
	nowindow.Set(cmd)
	return cmd.Output()
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

// LoggedIn reports whether requests carry a gh token, which raises GitHub's rate limit.
func (a *Auth) LoggedIn(ctx context.Context) bool { return a.get(ctx) != "" }

// IsGitHubHost reports whether host is one that the gh token may be sent to.
func IsGitHubHost(host string) bool {
	return host == "github.com" || host == "api.github.com"
}

// Apply sets the bearer token on req when it targets GitHub itself over https and a token is available. A
// redirect to another host (release asset storage) drops the header in net/http; a same-host downgrade to
// http is refused before the token is attached.
func (a *Auth) Apply(req *http.Request) {
	if req.URL.Scheme != "https" || !IsGitHubHost(req.URL.Hostname()) {
		return
	}
	if t := a.get(req.Context()); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
}

// noDowngrade is hc, or a copy of it when it has no redirect policy, that refuses a redirect to plain http: net/http
// keeps the Authorization header across a same-host redirect, scheme change included.
func noDowngrade(hc *http.Client) *http.Client {
	if hc.CheckRedirect != nil {
		return hc
	}
	secured := *hc
	secured.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		if req.URL.Scheme != "https" && via[0].URL.Scheme == "https" {
			return fmt.Errorf("refusing a redirect to %s://", req.URL.Scheme)
		}
		return nil
	}
	return &secured
}
