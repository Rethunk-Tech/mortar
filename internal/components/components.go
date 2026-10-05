// Package components loads the signed component manifest and downloads its assets.
package components

import (
	"bytes"
	"cmp"
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
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

const (
	// ReleasesURL lists Mortar's releases. Published releases are immutable, so every signed manifest is its own
	// components-<serial> release and the newest one is the current manifest.
	ReleasesURL = "https://api.github.com/repos/Rethunk-Tech/mortar/releases?per_page=100"
	tagPrefix   = "components-"
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

// GameInfo is one game's catalog entry: the names and ids the stores and mod sites know it by, its loaders and its mod
// sources. How a game is launched or modded stays in its Go implementation; this is only what can change without a
// Mortar release.
type GameInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Enabled is false for a game listed as coming later.
	Enabled bool `json:"enabled"`
	// Marker is a file every install of the game holds, at its root or one "game" folder down.
	Marker string `json:"marker"`
	// R2modmanFolder is r2modman's own folder name for the game (the Thunderstore schema's internalFolderName), the
	// parent of its profiles folder.
	R2modmanFolder string `json:"r2modmanFolder,omitempty"`
	// Metadata names the SMAPI-derived features that apply to the game: smapi-updates, smapi-compat, stardew-dataset.
	Metadata []string `json:"metadata"`
	// Paths names folders and files outside the install by role (saves, startupPreferences).
	Paths map[string]PathTemplate `json:"paths,omitempty"`
	// Deploy is how the profile's files reach the game: redirect (the loader points the game at the profile's folder,
	// nothing is placed) or link-into-install (files are placed into the install for the launch and taken back after).
	Deploy string `json:"deploy"`
	// Targets are the places the game's mod files go.
	Targets []TargetDef  `json:"targets"`
	Stores  GameStores   `json:"stores"`
	Loaders []GameLoader `json:"loaders"`
	Sources []GameSource `json:"sources"`
}

// PathTemplate is a path per platform of the game build. Tokens: {appData} {localAppData} {localLow} {documents}
// {xdgConfig} {xdgData} {home} {install}; the runtime that runs the install gives them their folders.
type PathTemplate struct {
	Windows string `json:"windows,omitempty"`
	Linux   string `json:"linux,omitempty"`
	Darwin  string `json:"darwin,omitempty"`
}

// Deploy methods.
const (
	DeployRedirect = "redirect"
	DeployLink     = "link-into-install"
)

// TargetDef is a content target: a named place mod files go. Root is where it lives in the profile: {profileMods}
// (the profile's mods folder), {profile} (the profile's root) or a folder below {profile}. Install is where the
// deploy puts its files in the game: {install} or a folder below it, empty for a game that is redirected to Root.
// Writable targets receive copies because the game writes into them; what it writes returns to Root. MaxDepth caps,
// per lower-case file extension, how many folders deep a file may sit below the target.
type TargetDef struct {
	ID       string         `json:"id"`
	Root     string         `json:"root"`
	Install  string         `json:"install,omitempty"`
	Writable bool           `json:"writable,omitempty"`
	MaxDepth map[string]int `json:"maxDepth,omitempty"`
}

// Target returns the game's content target with the given id.
func (g GameInfo) Target(id string) (TargetDef, bool) {
	for _, t := range g.Targets {
		if t.ID == id {
			return t, true
		}
	}
	return TargetDef{}, false
}

// GameStores names a game to each store that sells it; a store that does not sell it is nil.
type GameStores struct {
	Steam  *SteamStore  `json:"steam,omitempty"`
	GOG    *GOGStore    `json:"gog,omitempty"`
	Lutris *LutrisStore `json:"lutris,omitempty"`
}

// SteamStore names a game to Steam.
type SteamStore struct {
	AppID string `json:"appId"`
}

// GOGStore names a game to GOG: its product id and the folder GOG installers give it.
type GOGStore struct {
	ProductID string `json:"productId"`
	Folder    string `json:"folder"`
}

// LutrisStore names a game to Lutris: its slug and a word its executable paths contain.
type LutrisStore struct {
	Slug    string `json:"slug"`
	Keyword string `json:"keyword"`
}

// GameLoader is a mod loader a game can run.
type GameLoader struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// NexusModID is the loader's Nexus mod page, which Mortar installs outside any profile entry's Nexus source but
	// which is installed all the same.
	NexusModID int `json:"nexusModId,omitempty"`
	// Companion names the game's component (kind bridge) that Mortar installs alongside the loader so the running game
	// can talk to Mortar.
	Companion string `json:"companion,omitempty"`
}

// GameSource is a site the game's mods come from. Key is the site's name for the game (Nexus domain, Thunderstore
// community) and GameID its numeric id where the site's API uses one.
type GameSource struct {
	ID     string `json:"id"`
	Key    string `json:"key,omitempty"`
	GameID int    `json:"gameId,omitempty"`
}

// Source returns the game's source with the given id.
func (g GameInfo) Source(id string) (GameSource, bool) {
	for _, s := range g.Sources {
		if s.ID == id {
			return s, true
		}
	}
	return GameSource{}, false
}

// NexusDomain is the game's Nexus v1 URL segment, empty when Nexus does not host it.
func (g GameInfo) NexusDomain() string {
	s, _ := g.Source("nexus")
	return s.Key
}

// NexusID is the game's numeric Nexus id in the v2 API, 0 when Nexus does not host it.
func (g GameInfo) NexusID() int {
	s, _ := g.Source("nexus")
	return s.GameID
}

// LoaderNexusModID is the Nexus mod page of the game's first loader, 0 when it has none.
func (g GameInfo) LoaderNexusModID() int {
	if len(g.Loaders) == 0 {
		return 0
	}
	return g.Loaders[0].NexusModID
}

// SteamAppID is the game's Steam app id, empty when Steam does not sell it.
func (g GameInfo) SteamAppID() string {
	if g.Stores.Steam == nil {
		return ""
	}
	return g.Stores.Steam.AppID
}

// Validate checks that a game names itself, has a loader, and that no file or folder name could leave its folder.
func (g GameInfo) Validate() error {
	if g.ID == "" || g.Name == "" || g.Marker == "" {
		return errors.New("a game needs an id, a name and a marker file")
	}
	if len(g.Loaders) == 0 {
		return fmt.Errorf("game %q needs a loader", g.ID)
	}
	if g.Enabled && len(g.Targets) == 0 {
		return fmt.Errorf("enabled game %q needs a content target", g.ID)
	}
	if g.Enabled && g.Stores == (GameStores{}) {
		return fmt.Errorf("enabled game %q needs a store", g.ID)
	}
	names := []string{g.Marker}
	if g.Stores.GOG != nil {
		names = append(names, g.Stores.GOG.Folder)
	}
	for _, name := range names {
		if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
			return fmt.Errorf("game %q has an unsafe file or folder name %q", g.ID, name)
		}
	}
	for role, t := range g.Paths {
		for _, p := range []string{t.Windows, t.Linux, t.Darwin} {
			if p != "" && !strings.HasPrefix(p, "{") {
				return fmt.Errorf("game %q path %q must start with a token", g.ID, role)
			}
		}
	}
	if g.Deploy != DeployRedirect && g.Deploy != DeployLink {
		return fmt.Errorf("game %q has unknown deploy method %q", g.ID, g.Deploy)
	}
	targets := make(map[string]struct{}, len(g.Targets))
	for _, t := range g.Targets {
		if t.ID == "" || !underToken(t.Root, "{profileMods}", "{profile}") {
			return fmt.Errorf("game %q has a target without an id or with an unknown root %q", g.ID, t.Root)
		}
		if (t.Install == "") != (g.Deploy == DeployRedirect) || (t.Install != "" && !underToken(t.Install, "{install}")) {
			return fmt.Errorf("game %q target %q has install root %q, which its deploy method %q does not take", g.ID, t.ID, t.Install, g.Deploy)
		}
		if _, ok := targets[t.ID]; ok {
			return fmt.Errorf("game %q lists target %q more than once", g.ID, t.ID)
		}
		targets[t.ID] = struct{}{}
	}
	sources := make(map[string]struct{}, len(g.Sources))
	for _, s := range g.Sources {
		if s.ID == "" {
			return fmt.Errorf("game %q has a source without an id", g.ID)
		}
		if _, ok := sources[s.ID]; ok {
			return fmt.Errorf("game %q lists source %q more than once", g.ID, s.ID)
		}
		sources[s.ID] = struct{}{}
	}
	return nil
}

// underToken reports a path that is one of the tokens or a folder below one.
func underToken(p string, tokens ...string) bool {
	for _, t := range tokens {
		if p == t {
			return true
		}
		if rest, ok := strings.CutPrefix(p, t+"/"); ok && rest != "" && !strings.Contains(rest, "..") && !strings.Contains(rest, "\\") {
			return true
		}
	}
	return false
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
	HTTP *http.Client
	// ManifestURL, when set, is fetched as is instead of looking up the newest components release.
	ManifestURL   string
	ReleasesURL   string
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

func (c *Client) manifestURL(ctx context.Context) (string, error) {
	if c.ManifestURL != "" {
		return c.ManifestURL, nil
	}
	body, err := c.get(ctx, cmp.Or(c.ReleasesURL, ReleasesURL), maxManifest)
	if err != nil {
		return "", err
	}
	var releases []struct {
		Tag    string `json:"tag_name"`
		Draft  bool   `json:"draft"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", err
	}
	for _, r := range releases {
		if r.Draft || !strings.HasPrefix(r.Tag, tagPrefix) {
			continue
		}
		for _, a := range r.Assets {
			if a.Name == "components.json" {
				return a.URL, nil
			}
		}
	}
	return "", errors.New("no components release is published")
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

// BundledGameByNexusDomain returns the game whose Nexus source key (the v1 URL segment) is domain, from the manifest
// compiled into Mortar; the browser extension names games by that domain.
func BundledGameByNexusDomain(domain string) (GameInfo, bool) {
	m, err := BundledManifest()
	if err != nil {
		return GameInfo{}, false
	}
	for _, g := range m.Games {
		if key := g.NexusDomain(); key != "" && strings.EqualFold(key, domain) {
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
	url, err := c.manifestURL(ctx)
	if err != nil {
		return nil, nil, err
	}
	manifest, err := c.get(ctx, url, maxManifest)
	if err != nil {
		return nil, nil, err
	}
	signature, err := c.get(ctx, url+".sig", maxSignature)
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

// HasMetadata reports whether the game uses the named metadata feature.
func (g GameInfo) HasMetadata(name string) bool { return slices.Contains(g.Metadata, name) }
