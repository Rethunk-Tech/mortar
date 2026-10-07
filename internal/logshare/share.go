// Package logshare uploads a SMAPI log to smapi.io's parser (POST /log, form field input).
package logshare

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/launch"
)

const (
	DefaultURL = "https://smapi.io/log"
	Timeout    = 60 * time.Second
)

var (
	ErrEmpty  = errors.New("the log is empty")
	ErrNoLink = errors.New("smapi.io did not return a log link")
	ErrLarge  = errors.New("the log is too large to upload")
)

// Client posts a log to the SMAPI parser and returns the public view URL.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New() *Client {
	return &Client{
		BaseURL: DefaultURL,
		HTTP:    &http.Client{Timeout: Timeout, CheckRedirect: noFollow},
	}
}

func noFollow(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// Upload posts the raw SMAPI log as application/x-www-form-urlencoded input=… and
// returns the created parse URL (https://smapi.io/log/{id}). Success is a redirect to /log/{id};
// smapi.io answers a failed or empty upload with HTTP 200 and its page.
func (c *Client) Upload(ctx context.Context, logText string) (string, error) {
	if strings.TrimSpace(logText) == "" {
		return "", ErrEmpty
	}
	if len(logText) > launch.MaxLogBytes {
		return "", ErrLarge
	}
	base := cmp.Or(c.BaseURL, DefaultURL)
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: Timeout, CheckRedirect: noFollow}
	}
	form := url.Values{"input": {logText}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base, strings.NewReader(form))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 300 || res.StatusCode > 399 {
		if res.StatusCode == http.StatusOK {
			return "", errors.New("smapi.io did not accept the log")
		}
		return "", fmt.Errorf("smapi.io answered %s", res.Status)
	}
	loc := res.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("%w (HTTP %s)", ErrNoLink, res.Status)
	}
	abs, err := res.Request.URL.Parse(loc)
	if err != nil {
		return "", err
	}
	id, ok := strings.CutPrefix(abs.Path, "/log/")
	if !ok || id == "" || strings.Contains(id, "/") {
		return "", fmt.Errorf("%w (redirected to %q)", ErrNoLink, abs.Path)
	}
	return abs.Scheme + "://" + abs.Host + "/log/" + url.PathEscape(id), nil
}
