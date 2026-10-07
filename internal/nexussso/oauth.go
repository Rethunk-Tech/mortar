package nexussso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/secret"
)

const (
	// AuthBase is Nexus's OAuth2 server.
	AuthBase = "https://users.nexusmods.com/oauth"
	// Scope is public, which authorises the REST and GraphQL calls, with openid and profile to identify the user.
	Scope = "public openid profile"
	// RedirectPort is the one loopback port the sign-in listens on. Nexus registers the callback as a fixed URI, so
	// there is no fallback to another port. It sits in the IANA dynamic range, which no service owns.
	RedirectPort = 51762
	// CallbackPath is the path Nexus redirects to.
	CallbackPath = "/oauth/callback"
	// RedirectURI is the callback registered with Nexus.
	RedirectURI  = "http://127.0.0.1:51762" + CallbackPath
	redirectAddr = "127.0.0.1:51762"

	keyringItem   = "nexus-oauth"
	refreshMargin = time.Minute
)

// ErrPortBusy means the fixed callback port could not be opened.
var ErrPortBusy error = portBusy{}

type portBusy struct{}

func (portBusy) Error() string {
	return fmt.Sprintf("Port %d is in use; close the app using it and try again", RedirectPort)
}

// Tokens are the OAuth2 grant; the access token goes in the Authorization header of every Nexus call.
type Tokens struct {
	Access  string    `json:"access"`
	Refresh string    `json:"refresh"`
	Expires time.Time `json:"expires"`
}

// OAuth configures one PKCE run. Zero values use the production endpoints and timeout.
type OAuth struct {
	ClientID    string
	AuthBase    string
	Timeout     time.Duration
	HTTP        *http.Client
	OpenBrowser func(url string) error
	OnState     func(State)
	Now         func() time.Time
}

func (o OAuth) base() string {
	if o.AuthBase != "" {
		return o.AuthBase
	}
	return AuthBase
}

func (o OAuth) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o OAuth) client() *http.Client {
	if o.HTTP != nil {
		return o.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

// Authorize runs the loopback authorization-code flow and returns the tokens.
func (o OAuth) Authorize(ctx context.Context) (Tokens, error) {
	if o.ClientID == "" {
		return Tokens{}, ErrNoSlug
	}
	timeout := o.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ln, err := (&net.ListenConfig{}).Listen(runCtx, "tcp", redirectAddr)
	if err != nil {
		return Tokens{}, ErrPortBusy
	}
	redirect := RedirectURI
	verifier := rand.Text() + rand.Text()
	state := rand.Text()
	sum := sha256.Sum256([]byte(verifier))

	type result struct{ code, err string }
	got := make(chan result, 1)
	srv := &http.Server{ReadHeaderTimeout: 10 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != CallbackPath || q.Get("state") != state {
			http.NotFound(w, r)
			return
		}
		res := result{code: q.Get("code"), err: q.Get("error")}
		if res.code == "" && res.err == "" {
			res.err = "no code returned"
		}
		_, _ = w.Write([]byte("Signed in. You can close this tab and return to Mortar."))
		select {
		case got <- res:
		default:
		}
	})}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {o.ClientID},
		"redirect_uri":          {redirect},
		"scope":                 {Scope},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
	}
	if o.OpenBrowser == nil {
		return Tokens{}, errors.New("no browser to open")
	}
	if err := o.OpenBrowser(o.base() + "/authorize?" + q.Encode()); err != nil {
		return Tokens{}, err
	}
	if o.OnState != nil {
		o.OnState(Waiting)
	}
	select {
	case res := <-got:
		if res.err != "" {
			return Tokens{}, fmt.Errorf("nexus sign-in failed: %s", res.err)
		}
		return o.exchange(runCtx, url.Values{
			"grant_type": {"authorization_code"}, "code": {res.code}, "redirect_uri": {redirect}, "code_verifier": {verifier},
		})
	case <-runCtx.Done():
		if ctx.Err() != nil {
			return Tokens{}, ctx.Err()
		}
		return Tokens{}, ErrTimeout
	}
}

func (o OAuth) exchange(ctx context.Context, form url.Values) (Tokens, error) {
	form.Set("client_id", o.ClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base()+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		return Tokens{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := o.client().Do(req)
	if err != nil {
		return Tokens{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Tokens{}, fmt.Errorf("nexus token endpoint answered %s", resp.Status)
	}
	var raw struct {
		Access    string `json:"access_token"`
		Refresh   string `json:"refresh_token"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Tokens{}, err
	}
	if raw.Access == "" {
		return Tokens{}, errors.New("nexus token endpoint returned no access token")
	}
	return Tokens{Access: raw.Access, Refresh: raw.Refresh, Expires: o.now().Add(time.Duration(raw.ExpiresIn) * time.Second)}, nil
}

// Save keeps the tokens in the OS keyring.
func Save(t Tokens) error {
	b, _ := json.Marshal(t)
	return secret.Set(keyringItem, string(b))
}

// Load returns the stored tokens, or secret.ErrNotFound.
func Load() (Tokens, error) {
	raw, err := secret.Get(keyringItem)
	if err != nil {
		return Tokens{}, err
	}
	var t Tokens
	return t, json.Unmarshal([]byte(raw), &t)
}

// Forget deletes the stored tokens.
func Forget() error { return secret.Delete(keyringItem) }

// Fresh returns stored tokens, refreshing and re-saving them when they expire within a minute.
func (o OAuth) Fresh(ctx context.Context) (Tokens, error) {
	t, err := Load()
	if err != nil {
		return Tokens{}, err
	}
	if o.now().Add(refreshMargin).Before(t.Expires) {
		return t, nil
	}
	if t.Refresh == "" {
		return Tokens{}, errors.New("nexus access token expired and no refresh token is stored")
	}
	next, err := o.exchange(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {t.Refresh}})
	if err != nil {
		return Tokens{}, err
	}
	if next.Refresh == "" {
		next.Refresh = t.Refresh
	}
	return next, Save(next)
}
