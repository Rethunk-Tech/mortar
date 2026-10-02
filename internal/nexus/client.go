// Package nexus is a client for the Nexus Mods API. It sends a truthful application identity on every request,
// tracks the rate-limit headers and refuses to send once the remaining budget is nearly gone.
package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
)

const (
	// BaseURL is the public API.
	BaseURL = "https://api.nexusmods.com"
	// Game is the domain name of the only game Mortar supports so far.
	Game = "stardewvalley"
	// LimitFloor is the remaining-call count at or under which requests are refused until the window resets.
	LimitFloor = 5

	requestTimeout = 20 * time.Second
	maxBody        = 16 << 20
)

var (
	// ErrUnauthorized means the API key was rejected.
	ErrUnauthorized = errors.New("nexus rejected the API key")
	// ErrPremiumRequired means a download link needs the key and expires from an nxm:// link, or a premium account.
	ErrPremiumRequired = errors.New("nexus requires a premium account or an nxm:// link for this download")
	// ErrQuarantined means Nexus will not allow the file to be downloaded.
	ErrQuarantined = errors.New("nexus file quarantined")
)

// RateLimitError is a refused or rejected call; Reset is when the exhausted window renews.
type RateLimitError struct{ Reset time.Time }

func (e *RateLimitError) Error() string {
	return "nexus rate limit reached until " + e.Reset.Format(time.RFC3339)
}

// StatusError is any other non-200 answer.
type StatusError struct {
	Code   int
	Status string
}

func (e *StatusError) Error() string { return "nexus answered " + e.Status }

// Window is one rate-limit window as last reported by the API.
type Window struct {
	Remaining int       `json:"remaining"`
	Limit     int       `json:"limit"`
	Reset     time.Time `json:"reset"`
}

// Limits is the daily and hourly budget; Known is false until a response has carried the headers.
type Limits struct {
	Known  bool   `json:"known"`
	Daily  Window `json:"daily"`
	Hourly Window `json:"hourly"`
}

type limiter struct {
	mu sync.Mutex
	v  Limits
}

// Client talks to the API with one user's key. Copies made by WithKey share the rate-limit state.
type Client struct {
	HTTP *http.Client
	// BaseURL and CacheDir default to the public API and <datadir>/cache/nexus; tests override them.
	BaseURL  string
	CacheDir string
	Now      func() time.Time
	key      string
	version  string
	lim      *limiter
	track    *trackedCache
	scanMu   *sync.Mutex
	scans    map[int]map[int]string
	onLimits func(Limits)
}

// New returns a client that identifies itself as Mortar version.
func New(version string) *Client {
	return &Client{version: version, lim: &limiter{}, track: &trackedCache{}, scanMu: &sync.Mutex{}, scans: map[int]map[int]string{}}
}

// WithKey returns a client that authenticates with key and shares c's rate-limit and tracked-list state.
func (c *Client) WithKey(key string) *Client {
	return &Client{HTTP: c.HTTP, BaseURL: c.BaseURL, CacheDir: c.CacheDir, Now: c.Now, key: key, version: c.version, lim: c.lim, track: c.track, scanMu: c.scanMu, scans: c.scans, onLimits: c.onLimits}
}

// SetLimitsHook is called after a response updates the rate-limit budget.
func (c *Client) SetLimitsHook(fn func(Limits)) {
	c.onLimits = fn
}

// Limits returns the budget from the latest response.
func (c *Client) Limits() Limits {
	c.lim.mu.Lock()
	defer c.lim.mu.Unlock()
	return c.lim.v
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now().UTC()
}

func (c *Client) cacheDir() (string, error) {
	if c.CacheDir != "" {
		return c.CacheDir, nil
	}
	base, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cache", "nexus"), nil
}

// blocked returns the error to refuse with when a window is at the floor and has not reset yet.
func (c *Client) blocked() error {
	l := c.Limits()
	if !l.Known {
		return nil
	}
	now := c.now()
	for _, w := range []Window{l.Daily, l.Hourly} {
		if w.Remaining <= LimitFloor && now.Before(w.Reset) {
			return &RateLimitError{Reset: w.Reset}
		}
	}
	return nil
}

func (c *Client) record(h http.Header) {
	get := func(name string) (int, bool) {
		n, err := strconv.Atoi(h.Get(name))
		return n, err == nil
	}
	dr, ok1 := get("X-Rl-Daily-Remaining")
	hr, ok2 := get("X-Rl-Hourly-Remaining")
	if !ok1 || !ok2 {
		return
	}
	dl, _ := get("X-Rl-Daily-Limit")
	hl, _ := get("X-Rl-Hourly-Limit")
	v := Limits{
		Known:  true,
		Daily:  Window{Remaining: dr, Limit: dl, Reset: parseReset(h.Get("X-Rl-Daily-Reset"))},
		Hourly: Window{Remaining: hr, Limit: hl, Reset: parseReset(h.Get("X-Rl-Hourly-Reset"))},
	}
	c.lim.mu.Lock()
	c.lim.v = v
	c.lim.mu.Unlock()
	// The hook runs unlocked: it reads the budget back through Limits, which takes the same lock.
	if c.onLimits != nil {
		c.onLimits(v)
	}
}

func parseReset(s string) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05 -0700", "2006-01-02 15:04:05 MST"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// get decodes a 200 JSON answer for path into out; map403 turns a 403 into ErrPremiumRequired.
func (c *Client) get(ctx context.Context, path string, map403 bool, out any) error {
	if err := c.blocked(); err != nil {
		return err
	}
	base := c.BaseURL
	if base == "" {
		base = BaseURL
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Apikey", c.key)
	req.Header.Set("Application-Name", "Mortar")
	req.Header.Set("Application-Version", c.version)
	req.Header.Set("User-Agent", "Mortar/"+c.version)
	req.Header.Set("Accept", "application/json")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	c.record(resp.Header)
	switch {
	case resp.StatusCode == http.StatusOK:
	case resp.StatusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case resp.StatusCode == http.StatusForbidden && map403:
		return ErrPremiumRequired
	case resp.StatusCode == http.StatusTooManyRequests:
		reset := c.Limits().Hourly.Reset
		if d := c.Limits().Daily; d.Remaining <= 0 {
			reset = d.Reset
		}
		return &RateLimitError{Reset: reset}
	default:
		return &StatusError{Code: resp.StatusCode, Status: resp.Status}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// User is the account a key belongs to.
type User struct {
	ID        int    `json:"userId"`
	Name      string `json:"name"`
	IsPremium bool   `json:"isPremium"`
}

// Validate identifies the account behind the client's key.
func (c *Client) Validate(ctx context.Context) (User, error) {
	var raw struct {
		ID        int    `json:"user_id"`
		Name      string `json:"name"`
		IsPremium bool   `json:"is_premium"`
	}
	if err := c.get(ctx, "/v1/users/validate.json", false, &raw); err != nil {
		return User{}, err
	}
	return User{ID: raw.ID, Name: raw.Name, IsPremium: raw.IsPremium}, nil
}

// File is one downloadable file of a mod; Category is empty when Nexus reports null. Name is its title on the
// page and Description its BBCode.
type File struct {
	FileID      int       `json:"fileId"`
	FileName    string    `json:"fileName"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	ModVersion  string    `json:"modVersion"`
	Category    string    `json:"category"`
	SizeKB      int64     `json:"sizeKb"`
	IsPrimary   bool      `json:"isPrimary"`
	Uploaded    time.Time `json:"uploaded"`
	// ReplacedBy is the file the author uploaded as this one's update (Nexus file_updates), or 0.
	ReplacedBy int `json:"replacedBy"`
}

const stardewValleyGameID = 1303

// ScanStatuses returns Nexus's v2 virus-scan status for each file. Results are cached for this client.
func (c *Client) ScanStatuses(ctx context.Context, modID int) (map[int]string, error) {
	c.scanMu.Lock()
	if statuses, ok := c.scans[modID]; ok {
		c.scanMu.Unlock()
		return statuses, nil
	}
	c.scanMu.Unlock()

	body, err := json.Marshal(map[string]string{
		"query": fmt.Sprintf("{ modFiles(modId: %d, gameId: %d) { fileId scannedV2 } }", modID, stardewValleyGameID),
	})
	if err != nil {
		return nil, err
	}
	base := c.BaseURL
	if base == "" {
		base = BaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v2/graphql", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Apikey", c.key)
	req.Header.Set("Application-Name", "Mortar")
	req.Header.Set("Application-Version", c.version)
	req.Header.Set("User-Agent", "Mortar/"+c.version)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{Code: resp.StatusCode, Status: resp.Status}
	}
	var raw struct {
		Data struct {
			Files []struct {
				FileID  int    `json:"fileId"`
				Scanned string `json:"scannedV2"`
			} `json:"modFiles"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&raw); err != nil {
		return nil, err
	}
	statuses := make(map[int]string, len(raw.Data.Files))
	for _, file := range raw.Data.Files {
		statuses[file.FileID] = file.Scanned
	}
	c.scanMu.Lock()
	c.scans[modID] = statuses
	c.scanMu.Unlock()
	return statuses, nil
}

// Files lists every file of a mod.
func (c *Client) Files(ctx context.Context, modID int) ([]File, error) {
	var raw struct {
		Files []struct {
			FileID      int       `json:"file_id"`
			FileName    string    `json:"file_name"`
			Name        string    `json:"name"`
			Description string    `json:"description"`
			Version     string    `json:"version"`
			ModVersion  string    `json:"mod_version"`
			Category    *string   `json:"category_name"`
			SizeKB      int64     `json:"size_kb"`
			IsPrimary   bool      `json:"is_primary"`
			Uploaded    time.Time `json:"uploaded_time"`
		} `json:"files"`
		Updates []struct {
			Old int `json:"old_file_id"`
			New int `json:"new_file_id"`
		} `json:"file_updates"`
	}
	if err := c.get(ctx, fmt.Sprintf("/v1/games/%s/mods/%d/files.json", Game, modID), false, &raw); err != nil {
		return nil, err
	}
	files := make([]File, 0, len(raw.Files))
	for _, f := range raw.Files {
		file := File{
			FileID: f.FileID, FileName: f.FileName, Name: f.Name, Description: f.Description, Version: f.Version,
			ModVersion: f.ModVersion, SizeKB: f.SizeKB, IsPrimary: f.IsPrimary, Uploaded: f.Uploaded.UTC(),
		}
		if f.Category != nil {
			file.Category = *f.Category
		}
		files = append(files, file)
	}
	for _, u := range raw.Updates {
		for i := range files {
			if files[i].FileID == u.Old && u.New > files[i].ReplacedBy {
				files[i].ReplacedBy = u.New
			}
		}
	}
	return files, nil
}

// Link is one download mirror.
type Link struct {
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	URI       string `json:"uri"`
}

// DownloadLinks asks for a file's download mirrors. Premium accounts pass no key; a free account passes the key
// and expires of an nxm:// link and otherwise gets ErrPremiumRequired.
func (c *Client) DownloadLinks(ctx context.Context, modID, fileID int, key string, expires int64) ([]Link, error) {
	path := fmt.Sprintf("/v1/games/%s/mods/%d/files/%d/download_link.json", Game, modID, fileID)
	if key != "" {
		path += "?" + url.Values{"key": {key}, "expires": {strconv.FormatInt(expires, 10)}}.Encode()
	}
	var raw []struct {
		Name      string `json:"name"`
		ShortName string `json:"short_name"`
		URI       string `json:"URI"`
	}
	if err := c.get(ctx, path, true, &raw); err != nil {
		return nil, err
	}
	links := make([]Link, len(raw))
	for i, l := range raw {
		links[i] = Link{Name: l.Name, ShortName: l.ShortName, URI: l.URI}
	}
	return applyDownloadPreferences(links), nil
}
