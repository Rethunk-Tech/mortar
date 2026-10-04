// Package components loads the signed component manifest and downloads its assets.
package components

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/github"
	"github.com/Rethunk-AI/mortar/internal/meta"
)

const (
	ManifestURL = "https://github.com/Rethunk-AI/mortar/releases/download/components/components.json"
	cacheName   = "components-manifest.json"
	day         = 24 * time.Hour

	maxManifest  = 8 << 20
	maxSignature = 1 << 20
	maxComponent = 2 << 30
)

//go:embed components.json
var bundledJSON []byte

// Source identifies the host and project that publishes a component.
type Source struct {
	Host      string `json:"host"`
	Owner     string `json:"owner,omitempty"`
	Repo      string `json:"repo,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// SourceComponent is one entry in components.source.json.
type SourceComponent struct {
	Game         string `json:"game"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Source       Source `json:"source"`
	AssetPattern string `json:"asset"`
	Version      string `json:"version"`
	Accepted     string `json:"accepted,omitempty"`
}

// SourceFile is the committed input read by cmd/components.
type SourceFile struct {
	Components []SourceComponent `json:"components"`
	Games      []GameInfo        `json:"games"`
}

// GameInfo is one game's identity: the names and ids the stores and mod sites know it by. How a game is launched or
// modded stays in its Go implementation; this is only what can change without a Mortar release.
type GameInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SteamAppID string `json:"steamAppId"`
	// Marker is a file every install of the game holds, at its root or one "game" folder down.
	Marker string     `json:"marker"`
	Loader string     `json:"loader"`
	GOG    GOGInfo    `json:"gog"`
	Lutris LutrisInfo `json:"lutris"`
	Nexus  NexusInfo  `json:"nexus"`
}

// GOGInfo names a game to GOG: its product id and the folder GOG installers give it.
type GOGInfo struct {
	ProductID string `json:"productId"`
	Folder    string `json:"folder"`
}

// LutrisInfo names a game to Lutris: its slug and a word its executable paths contain.
type LutrisInfo struct {
	Slug    string `json:"slug"`
	Keyword string `json:"keyword"`
}

// NexusInfo names a game to Nexus Mods: its domain in v1 URLs and its numeric id in the v2 API.
type NexusInfo struct {
	Domain string `json:"domain"`
	ID     int    `json:"id"`
	// LoaderModID is the Nexus mod page of the game's mod loader, which Mortar installs outside any profile entry's
	// Nexus source but which is installed all the same.
	LoaderModID int `json:"loaderModId,omitempty"`
}

// Validate checks that a game names itself and that no file or folder name could leave its folder.
func (g GameInfo) Validate() error {
	if g.ID == "" || g.Name == "" || g.Marker == "" {
		return errors.New("a game needs an id, a name and a marker file")
	}
	for _, name := range []string{g.Marker, g.GOG.Folder} {
		if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
			return fmt.Errorf("game %q has an unsafe file or folder name %q", g.ID, name)
		}
	}
	return nil
}

// Component is one resolved, hashed asset in a manifest.
type Component struct {
	Game     string `json:"game"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Source   Source `json:"source"`
	Tag      string `json:"tag"`
	Asset    string `json:"asset"`
	Version  string `json:"version"`
	Accepted string `json:"accepted,omitempty"`
	SHA256   string `json:"sha256"`
}

// Manifest is the signed set of component assets Mortar may install.
type Manifest struct {
	Serial     uint64      `json:"serial"`
	Components []Component `json:"components"`
	Games      []GameInfo  `json:"games"`
}

// Validate checks the source host, component identity and asset digest.
func (s Source) Validate() error {
	switch strings.ToLower(strings.TrimSpace(s.Host)) {
	case "github.com":
		if !validPart(s.Owner) || !validPart(s.Repo) {
			return errors.New("GitHub component source needs an owner and repository")
		}
	case "thunderstore.io":
		if !validPart(s.Namespace) || !validPart(s.Name) {
			return errors.New("thunderstore component source needs a namespace and name")
		}
	default:
		return fmt.Errorf("component source host %q is not allowed", s.Host)
	}
	return nil
}

func validPart(value string) bool {
	return value != "" && !strings.ContainsAny(value, "/\\?#")
}

// Validate checks a resolved component.
func (c Component) Validate() error {
	if c.Game == "" || c.Name == "" {
		return errors.New("component game and name are required")
	}
	if c.Kind != "loader" && c.Kind != "bridge" {
		return fmt.Errorf("component %q has unknown kind %q", c.Name, c.Kind)
	}
	if c.Version == "" || c.Asset == "" || c.Tag == "" {
		return fmt.Errorf("component %q is missing its resolved release", c.Name)
	}
	if err := c.Source.Validate(); err != nil {
		return fmt.Errorf("component %q: %w", c.Name, err)
	}
	if len(c.SHA256) != sha256.Size*2 {
		return fmt.Errorf("component %q has an invalid sha256", c.Name)
	}
	if _, err := hex.DecodeString(c.SHA256); err != nil {
		return fmt.Errorf("component %q has an invalid sha256: %w", c.Name, err)
	}
	return nil
}

// Validate checks every component and rejects duplicate game/name pairs.
func (m Manifest) Validate() error {
	if m.Serial == 0 {
		return errors.New("component manifest has no serial")
	}
	seen := make(map[string]struct{}, len(m.Components))
	for _, c := range m.Components {
		if err := c.Validate(); err != nil {
			return err
		}
		key := c.Game + "\x00" + c.Name
		if _, ok := seen[key]; ok {
			return fmt.Errorf("component %q is listed more than once for %s", c.Name, c.Game)
		}
		seen[key] = struct{}{}
	}
	games := make(map[string]struct{}, len(m.Games))
	for _, g := range m.Games {
		if err := g.Validate(); err != nil {
			return err
		}
		if _, ok := games[g.ID]; ok {
			return fmt.Errorf("game %q is listed more than once", g.ID)
		}
		games[g.ID] = struct{}{}
	}
	return nil
}

// Decode parses and validates a manifest after its signature has been checked.
func Decode(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("decode component manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// Verify checks a detached Ed25519 signature without parsing the manifest.
func Verify(manifest, signature, publicKey []byte) error {
	key, err := parsePublicKey(publicKey)
	if err != nil {
		return err
	}
	sig, err := parseSignature(signature)
	if err != nil {
		return err
	}
	if !ed25519.Verify(key, manifest, sig) {
		return errors.New("component manifest signature is invalid")
	}
	return nil
}

func parsePublicKey(data []byte) (ed25519.PublicKey, error) {
	if len(data) == ed25519.PublicKeySize {
		return ed25519.PublicKey(data), nil
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("component public key is not PEM")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse component public key: %w", err)
	}
	public, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("component public key is not Ed25519")
	}
	return public, nil
}

func parseSignature(data []byte) ([]byte, error) {
	if len(data) == ed25519.SignatureSize {
		return data, nil
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == ed25519.SignatureSize {
		return trimmed, nil
	}
	for _, encoding := range []*base64.Encoding{base64.RawStdEncoding, base64.StdEncoding} {
		sig, err := encoding.DecodeString(string(trimmed))
		if err == nil && len(sig) == ed25519.SignatureSize {
			return sig, nil
		}
	}
	return nil, errors.New("component signature is not an Ed25519 signature")
}

// BundledManifest returns the manifest compiled into Mortar.
func BundledManifest() (Manifest, error) {
	return Decode(bundledJSON)
}

type cachedManifest struct {
	Manifest  []byte `json:"manifest"`
	Signature []byte `json:"signature"`
}

// Client loads the signed manifest and downloads assets named by it.
type Client struct {
	HTTP          *http.Client
	ManifestURL   string
	GitHubBaseURL string
	manifest      Manifest
}

// NewClient returns a component client using hc for manifest and asset requests.
func NewClient(hc *http.Client) *Client {
	return &Client{HTTP: hc}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) manifestURL() string {
	if c.ManifestURL != "" {
		return c.ManifestURL
	}
	return ManifestURL
}

// SetManifest selects the already verified manifest used by Component and Download.
func (c *Client) SetManifest(m Manifest) {
	c.manifest = m
}

// Manifest returns the currently selected manifest.
func (c *Client) Manifest() Manifest {
	return c.manifest
}

// Component returns a component by game and name.
func (c *Client) Component(game, name string) (Component, bool) {
	if component, ok := findComponent(c.manifest.Components, game, name); ok {
		return component, true
	}
	// Before Load (and when the fetched manifest lacks it) the manifest compiled into Mortar answers, as Game does,
	// so startup work that runs ahead of the network fetch still finds its components.
	m, err := BundledManifest()
	if err != nil {
		return Component{}, false
	}
	return findComponent(m.Components, game, name)
}

func findComponent(components []Component, game, name string) (Component, bool) {
	for _, component := range components {
		if component.Game == game && component.Name == name {
			return component, true
		}
	}
	return Component{}, false
}

// Game returns a game's identity from the selected manifest, else from the bundled one, so a manifest that lacks a
// game never leaves Mortar without it.
func (c *Client) Game(id string) (GameInfo, bool) {
	if g, ok := findGame(c.manifest.Games, id); ok {
		return g, true
	}
	return BundledGame(id)
}

// BundledGame returns a game's identity from the manifest compiled into Mortar.
func BundledGame(id string) (GameInfo, bool) {
	m, err := BundledManifest()
	if err != nil {
		return GameInfo{}, false
	}
	return findGame(m.Games, id)
}

// BundledGameByNexusDomain returns the game whose Nexus domain (the v1 URL segment) is domain, from the manifest
// compiled into Mortar; the browser extension names games by that domain.
func BundledGameByNexusDomain(domain string) (GameInfo, bool) {
	m, err := BundledManifest()
	if err != nil {
		return GameInfo{}, false
	}
	for _, g := range m.Games {
		if g.Nexus.Domain != "" && strings.EqualFold(g.Nexus.Domain, domain) {
			return g, true
		}
	}
	return GameInfo{}, false
}

func findGame(games []GameInfo, id string) (GameInfo, bool) {
	for _, g := range games {
		if g.ID == id {
			return g, true
		}
	}
	return GameInfo{}, false
}

// Load fetches the signed manifest at startup, using the existing meta cache for a daily TTL. A failed
// fetch or invalid cached entry falls back to the bundled manifest.
func (c *Client) Load(ctx context.Context, cache *meta.Client, publicKey []byte) (Manifest, error) {
	bundled, bundledErr := BundledManifest()
	if bundledErr != nil {
		return Manifest{}, bundledErr
	}
	// A fetched manifest older than the one this build ships or the one already cached is a rollback.
	oldSerial := bundled.Serial
	if cache != nil {
		if old, ok := meta.Peek[cachedManifest](cache, cacheName); ok {
			if err := Verify(old.Manifest, old.Signature, publicKey); err == nil {
				if parsed, err := Decode(old.Manifest); err == nil {
					oldSerial = max(oldSerial, parsed.Serial)
				}
			}
		}
	}
	var fetchFailure error
	fetch := func() (cachedManifest, error) {
		manifest, signature, err := c.fetch(ctx)
		if err != nil {
			fetchFailure = err
			return cachedManifest{}, err
		}
		if err := Verify(manifest, signature, publicKey); err != nil {
			fetchFailure = err
			return cachedManifest{}, err
		}
		parsed, err := Decode(manifest)
		if err != nil {
			fetchFailure = err
			return cachedManifest{}, err
		}
		if parsed.Serial < oldSerial {
			fetchFailure = fmt.Errorf("component manifest serial %d is older than cached serial %d", parsed.Serial, oldSerial)
			return cachedManifest{}, fetchFailure
		}
		return cachedManifest{Manifest: manifest, Signature: signature}, nil
	}

	var entry cachedManifest
	var err error
	if cache != nil {
		entry, err = meta.Cached(cache, cacheName, day, fetch)
	} else {
		entry, err = fetch()
	}
	if err == nil {
		if verifyErr := Verify(entry.Manifest, entry.Signature, publicKey); verifyErr == nil {
			if parsed, parseErr := Decode(entry.Manifest); parseErr == nil {
				c.SetManifest(parsed)
				return parsed, fetchFailure
			} else {
				err = parseErr
			}
		} else {
			err = verifyErr
		}
	}
	c.SetManifest(bundled)
	return bundled, err
}

func (c *Client) fetch(ctx context.Context) ([]byte, []byte, error) {
	manifest, err := c.get(ctx, c.manifestURL(), maxManifest)
	if err != nil {
		return nil, nil, err
	}
	signature, err := c.get(ctx, c.manifestURL()+".sig", maxSignature)
	if err != nil {
		return nil, nil, err
	}
	return manifest, signature, nil
}

func (c *Client) get(ctx context.Context, address string, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("component manifest request returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errors.New("component manifest response is too large")
	}
	return body, nil
}

// Download fetches a component into dest and verifies its SHA-256 before replacing dest. A failed or
// mismatched download never changes an existing dest.
func (c *Client) Download(ctx context.Context, component Component, dest string) error {
	if err := component.Validate(); err != nil {
		return err
	}
	if strings.ToLower(component.Source.Host) != "github.com" {
		return fmt.Errorf("component source host %q is not supported for downloads", component.Source.Host)
	}
	tag := component.Tag
	if tag == "" {
		tag = component.Version
	}
	base := c.GitHubBaseURL
	if base == "" {
		base = "https://github.com"
	}
	address := strings.TrimRight(base, "/") + "/" + component.Source.Owner + "/" + component.Source.Repo +
		"/releases/download/" + tag + "/" + component.Asset
	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".mortar-component-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	defer func() {
		_ = os.Remove(tmpPath)
		_ = os.Remove(github.ResumeSidecar(tmpPath))
	}()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := github.Download(ctx, c.HTTP, address, tmpPath, maxComponent, nil); err != nil {
		return fmt.Errorf("download component %s: %w", component.Name, err)
	}
	sum, err := fsx.SHA256(tmpPath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(sum, component.SHA256) {
		return fmt.Errorf("component %s has sha256 %s, want %s", component.Name, sum, component.SHA256)
	}
	return datadir.WriteStream(dest, 0o600, func(w io.Writer) error {
		f, err := fsx.Open(tmpPath)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		_, err = io.Copy(w, f)
		return err
	})
}
