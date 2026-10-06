// Package share encodes a profile as a shareable link and reads links back. Everything parsed here comes from
// outside: caps are enforced while reading, and nothing is trusted until it passes the checks.
package share

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/andybalholm/brotli"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

const (
	// FormatVersion is the payload's leading number; no other version is read.
	FormatVersion = 3

	MaxEncoded    = 8 << 10
	MaxDecoded    = 64 << 10
	MaxEntries    = 1000
	MaxNameLength = 60

	maxID = 1<<31 - 1
)

var (
	// ErrNotLink means the text is neither a Mortar link nor a bare payload.
	ErrNotLink = errors.New("not a Mortar share link")
	// ErrTooLarge means the encoded payload is over MaxEncoded or the decompressed one over MaxDecoded.
	ErrTooLarge = errors.New("share payload is too large")
	// ErrNewerVersion means the payload was made by a newer Mortar.
	ErrNewerVersion = errors.New("this share was made by a newer Mortar: update Mortar to open it")
	// ErrMalformed means the payload decoded but its contents are not a valid profile share.
	ErrMalformed = errors.New("share payload is malformed")
)

// Ref names one file to install: a Nexus mod file, a GitHub release asset as "<owner>/<repo>@<tag>/<asset>", or a
// Thunderstore package.
// On the wire each ref is an object whose "s" names its source.
type Ref struct {
	ModID  int    `json:"modId,omitempty"`
	FileID int    `json:"fileId,omitempty"`
	GitHub string `json:"github,omitempty"`
	// Package is a Thunderstore "Namespace-Name" at Version (the newest when Version is empty).
	Package  string                         `json:"package,omitempty"`
	Version  string                         `json:"version,omitempty"`
	Disabled []mod.ID                       `json:"disabled,omitempty"`
	Fomod    map[string]map[string][]string `json:"fomod,omitempty"`
	Note     string                         `json:"note,omitempty"`
	Tags     []string                       `json:"tags,omitempty"`
	Overlay  *Overlay                       `json:"overlay,omitempty"`
	// SizeKB and MinGame are for the share page alone: the mod's size on disk, rounded to two significant digits,
	// and the oldest game version its mods declare they run on.
	SizeKB  int64  `json:"sizeKb,omitempty"`
	MinGame string `json:"minGame,omitempty"`
	key     string
}

// Overlay places an optional file inside the main file of the same mod: the folder of its archive that is laid
// over, where in the main file's folder it goes (both slash paths, "" for the top), and whether it starts off.
type Overlay struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	Off  bool   `json:"off,omitempty"`
}

// Shared is what a link carries.
type Shared struct {
	// Game is the Mortar game id the share is for.
	Game string
	// SourceKeys maps a source id to that source's name for the game, e.g. {"nexus": "stardewvalley"}.
	SourceKeys map[string]string
	Name       string
	Entries    []Ref
	// GameVersion is the version of the game the profile was shared from, empty when unknown.
	GameVersion string
}

// LeftOut is an enabled entry that cannot travel in a link.
type LeftOut struct {
	Key    string
	Reason string
}

// Include is what a share or export carries besides enabled mods. Zero values
// mean omit; use DefaultInclude for today's behaviour.
type Include struct {
	DisabledMods bool `json:"disabledMods"`
	FomodChoices bool `json:"fomodChoices"`
	Notes        bool `json:"notes"`
	ConfigFiles  bool `json:"configFiles"`
}

// DefaultInclude is the registry default: disabled mods off, the rest on.
func DefaultInclude() Include {
	return Include{FomodChoices: true, Notes: true, ConfigFiles: true}
}

// Result is an encoded profile: the payload, both link forms, and the entries left out.
type Result struct {
	Shared  Shared
	Payload string
	Web     string
	App     string
	LeftOut []LeftOut
}

// GitHubParts splits a GitHub ref into its "owner/repo", tag and asset name.
func (r Ref) GitHubParts() (repo, tag, asset string) {
	repo, rest, _ := strings.Cut(r.GitHub, "@")
	tag, asset, _ = strings.Cut(rest, "/")
	return repo, tag, asset
}

var githubRef = regexp.MustCompile(`^[A-Za-z0-9-]{1,39}/[A-Za-z0-9._-]{1,100}@[A-Za-z0-9._+-]{1,100}/[A-Za-z0-9._+~()-]{1,200}$`)

func validName(name string) bool {
	n := utf8.RuneCountInString(name)
	return n > 0 && n <= MaxNameLength && strings.TrimSpace(name) == name &&
		!strings.ContainsFunc(name, unicode.IsControl)
}

var (
	packageRef = regexp.MustCompile(`^[A-Za-z0-9_]+-[A-Za-z0-9_]+$`)
	versionRef = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

func (r Ref) valid() bool {
	if r.Package != "" {
		return r.ModID == 0 && r.FileID == 0 && r.GitHub == "" && packageRef.MatchString(r.Package) &&
			(r.Version == "" || versionRef.MatchString(r.Version))
	}
	if r.GitHub != "" {
		if r.ModID != 0 || r.FileID != 0 || !githubRef.MatchString(r.GitHub) {
			return false
		}
		// All-dot names pass the pattern but would walk the API URL's path or the download's file name.
		repo, tag, asset := r.GitHubParts()
		owner, name, _ := strings.Cut(repo, "/")
		return !slices.ContainsFunc([]string{owner, name, tag, asset}, func(p string) bool { return strings.Trim(p, ".") == "" })
	}
	return r.ModID > 0 && r.ModID <= maxID && r.FileID > 0 && r.FileID <= maxID
}

// wireRef is a ref as it travels: the source id, that source's coordinates, then the extras.
type wireRef struct {
	Source   string                         `json:"s"`
	Mod      int                            `json:"mod,omitempty"`
	File     int                            `json:"file,omitempty"`
	Repo     string                         `json:"repo,omitempty"`
	Tag      string                         `json:"tag,omitempty"`
	Asset    string                         `json:"asset,omitempty"`
	NS       string                         `json:"ns,omitempty"`
	Name     string                         `json:"name,omitempty"`
	Version  string                         `json:"version,omitempty"`
	Disabled []mod.ID                       `json:"disabled,omitempty"`
	Fomod    map[string]map[string][]string `json:"fomod,omitempty"`
	Note     string                         `json:"note,omitempty"`
	Tags     []string                       `json:"tags,omitempty"`
	Overlay  *Overlay                       `json:"overlay,omitempty"`
	KB       int64                          `json:"kb,omitempty"`
	Min      string                         `json:"min,omitempty"`
}

// MarshalJSON writes the ref as its wire object.
func (r Ref) MarshalJSON() ([]byte, error) {
	w := wireRef{Disabled: r.Disabled, Fomod: r.Fomod, Note: r.Note, Tags: r.Tags, Overlay: r.Overlay, KB: r.SizeKB, Min: r.MinGame}
	switch {
	case r.Package != "":
		w.Source, w.Version = "thunderstore", r.Version
		w.NS, w.Name, _ = strings.Cut(r.Package, "-")
	case r.GitHub != "":
		w.Source = "github"
		w.Repo, w.Tag, w.Asset = r.GitHubParts()
	default:
		w.Source, w.Mod, w.File = "nexus", r.ModID, r.FileID
	}
	return json.Marshal(w)
}

// UnmarshalJSON reads the wire object, so a ref read back from JSON passes the same checks as one in a link.
func (r *Ref) UnmarshalJSON(raw []byte) error {
	v, err := parseRef(raw)
	*r = v
	return err
}

func parseRef(raw json.RawMessage) (Ref, error) {
	var w wireRef
	if err := json.Unmarshal(raw, &w); err != nil {
		return Ref{}, fmt.Errorf("%w: bad entry", ErrMalformed)
	}
	r := Ref{Disabled: w.Disabled, Fomod: w.Fomod, Note: w.Note, Tags: w.Tags, Overlay: w.Overlay, SizeKB: w.KB, MinGame: w.Min}
	switch {
	case w.Source == "thunderstore" && w.Mod == 0 && w.File == 0 && w.Repo == "" && w.NS != "" && w.Name != "":
		r.Package, r.Version = w.NS+"-"+w.Name, w.Version
	case w.Source == "nexus" && w.Repo == "" && w.Tag == "" && w.Asset == "":
		r.ModID, r.FileID = w.Mod, w.File
	case w.Source == "github" && w.Mod == 0 && w.File == 0 && w.Repo != "" && w.Tag != "" && w.Asset != "":
		r.GitHub = w.Repo + "@" + w.Tag + "/" + w.Asset
	default:
		return Ref{}, fmt.Errorf("%w: bad entry", ErrMalformed)
	}
	return r, nil
}

var (
	gameIDPattern = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)
	sourceKey     = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
	gameVersion   = regexp.MustCompile(`^(\d{1,9}(\.\d{1,9}){0,3})?$`)
)

const maxSourceKeys = 16

func validOrigin(s Shared) bool {
	if !gameIDPattern.MatchString(s.Game) || len(s.SourceKeys) > maxSourceKeys {
		return false
	}
	for id, key := range s.SourceKeys {
		if !gameIDPattern.MatchString(id) || !sourceKey.MatchString(key) {
			return false
		}
	}
	return true
}

func checkShared(s Shared) error {
	if !validName(s.Name) {
		return fmt.Errorf("%w: bad profile name", ErrMalformed)
	}
	if !validOrigin(s) || !gameVersion.MatchString(s.GameVersion) {
		return fmt.Errorf("%w: bad game or source keys", ErrMalformed)
	}
	if len(s.Entries) > MaxEntries {
		return fmt.Errorf("%w: more than %d entries", ErrMalformed, MaxEntries)
	}
	for _, r := range s.Entries {
		if !r.valid() || !validDetails(r) || (r.GitHub == "" && r.Package == "" && s.SourceKeys["nexus"] == "") ||
			(r.Package != "" && s.SourceKeys["thunderstore"] == "") {
			return fmt.Errorf("%w: bad entry", ErrMalformed)
		}
	}
	return nil
}

func validDetails(r Ref) bool {
	if len(r.Disabled) > MaxEntries || len(r.Fomod) > MaxEntries {
		return false
	}
	if !validEntryNote(r.Note) || !validEntryTags(r.Tags) || r.SizeKB < 0 || r.SizeKB > maxID || !gameVersion.MatchString(r.MinGame) {
		return false
	}
	if r.Overlay != nil && (r.GitHub != "" || r.Package != "" || !validOverlayPath(r.Overlay.From) || !validOverlayPath(r.Overlay.To)) {
		return false
	}
	for _, disabled := range r.Disabled {
		id := string(disabled)
		if id == "" || len(id) > 100 || strings.TrimSpace(id) != id || strings.ContainsFunc(id, unicode.IsControl) {
			return false
		}
	}
	for step, groups := range r.Fomod {
		if step == "" || len(step) > 100 || strings.ContainsFunc(step, unicode.IsControl) || len(groups) > MaxEntries {
			return false
		}
		for group, choices := range groups {
			if group == "" || len(group) > 100 || strings.ContainsFunc(group, unicode.IsControl) || len(choices) > MaxEntries {
				return false
			}
			for _, choice := range choices {
				if len(choice) > 200 || strings.ContainsFunc(choice, unicode.IsControl) {
					return false
				}
			}
		}
	}
	return true
}

// validOverlayPath accepts "" or a relative slash path of plain segments that stays inside its folder.
func validOverlayPath(p string) bool {
	if p == "" {
		return true
	}
	if len(p) > maxRelPath || strings.ContainsFunc(p, unicode.IsControl) || strings.Contains(p, `\`) || strings.Contains(p, ":") {
		return false
	}
	for seg := range strings.SplitSeq(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}
	return true
}

func validEntryNote(note string) bool {
	if utf8.RuneCountInString(note) > profile.MaxEntryNote {
		return false
	}
	return !strings.ContainsFunc(note, badNoteRune)
}

// badNoteRune allows the line breaks and tabs a profile note may hold and refuses every other control character.
func badNoteRune(r rune) bool {
	return unicode.IsControl(r) && r != '\n' && r != '\t'
}

func validEntryTags(tags []string) bool {
	if len(tags) > profile.MaxEntryTags {
		return false
	}
	seen := map[string]bool{}
	for _, raw := range tags {
		tag, ok := profile.CleanTag(raw)
		if !ok {
			return false
		}
		key := strings.ToLower(tag)
		if seen[key] {
			return false
		}
		seen[key] = true
	}
	return true
}

// ImportEntryNotes copies note and tags from a shared ref onto an entry, truncating the note and dropping invalid tags.
func ImportEntryNotes(e *profile.Entry, r Ref) {
	e.Note, e.Tags = importEntryNoteTags(r.Note, r.Tags)
}

func importEntryNoteTags(note string, tags []string) (string, []string) {
	note = strings.TrimSpace(note)
	if n := utf8.RuneCountInString(note); n > profile.MaxEntryNote {
		note = string([]rune(note)[:profile.MaxEntryNote])
	}
	note = strings.Map(func(r rune) rune {
		if badNoteRune(r) {
			return -1
		}
		return r
	}, note)
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, raw := range tags {
		tag, ok := profile.CleanTag(raw)
		if !ok {
			continue
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tag)
		if len(out) >= profile.MaxEntryTags {
			break
		}
	}
	if len(out) == 0 {
		out = nil
	}
	return note, out
}

// MatchesEntry reports whether the ref names the same mod file as the profile entry.
func (r Ref) MatchesEntry(e profile.Entry) bool {
	if r.Package != "" {
		return e.Source.Kind == profile.KindThunderstore && strings.EqualFold(e.Source.Name, r.Package) &&
			(r.Version == "" || e.Source.Version == r.Version)
	}
	if r.GitHub != "" {
		repo, tag, asset := r.GitHubParts()
		return e.Source.Kind == profile.KindGitHub && strings.EqualFold(e.Source.Repo, repo) &&
			e.Source.Tag == tag && e.Source.Asset == asset
	}
	return e.Source.Kind == profile.KindNexus && e.Source.ModID == r.ModID && e.Source.FileID == r.FileID
}

func cloneFomod(in map[string]map[string][]string) map[string]map[string][]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]map[string][]string, len(in))
	for step, groups := range in {
		out[step] = make(map[string][]string, len(groups))
		for group, choices := range groups {
			out[step][group] = slices.Clone(choices)
		}
	}
	return out
}

// versioned splits a payload document into its version and the rest, refusing a newer version before the rest
// is looked at, since its shape may differ.
// versioned reads the array, checks its version and that it has minParts to minParts+1 elements; the last one is
// optional.
func versioned(raw []byte, minParts int) ([]json.RawMessage, error) {
	var doc []json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil || len(doc) == 0 {
		return nil, fmt.Errorf("%w: not a JSON array", ErrMalformed)
	}
	var v int
	if err := json.Unmarshal(doc[0], &v); err != nil || v < 1 {
		return nil, fmt.Errorf("%w: bad version", ErrMalformed)
	}
	if v > FormatVersion {
		return nil, ErrNewerVersion
	}
	if v != FormatVersion {
		return nil, fmt.Errorf("%w: unsupported version %d", ErrMalformed, v)
	}
	if len(doc) != minParts && len(doc) != minParts+1 {
		return nil, fmt.Errorf("%w: wrong shape", ErrMalformed)
	}
	return doc, nil
}

func parseEntries(raw json.RawMessage) ([]Ref, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("%w: bad entries", ErrMalformed)
	}
	if len(items) > MaxEntries {
		return nil, fmt.Errorf("%w: more than %d entries", ErrMalformed, MaxEntries)
	}
	out := make([]Ref, 0, len(items))
	for _, it := range items {
		r, err := parseRef(it)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s Shared) payload() (string, error) {
	if err := checkShared(s); err != nil {
		return "", err
	}
	name, err := json.Marshal(s.Name)
	if err != nil {
		return "", err
	}
	entries, err := json.Marshal(s.Entries)
	if err != nil {
		return "", err
	}
	game, err := json.Marshal(s.Game)
	if err != nil {
		return "", err
	}
	if s.SourceKeys == nil {
		s.SourceKeys = map[string]string{}
	}
	keys, err := json.Marshal(s.SourceKeys)
	if err != nil {
		return "", err
	}
	raw := fmt.Appendf(nil, "[%d,%s,%s,%s,%s", FormatVersion, name, game, keys, entries)
	// The game version is the array's optional last element, left out when unknown.
	if s.GameVersion != "" {
		gv, err := json.Marshal(s.GameVersion)
		if err != nil {
			return "", err
		}
		raw = fmt.Appendf(raw, ",%s", gv)
	}
	raw = append(raw, ']')
	var buf bytes.Buffer
	w := brotli.NewWriterLevel(&buf, brotli.BestCompression)
	if _, err := w.Write(raw); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	out := base64.RawURLEncoding.EncodeToString(buf.Bytes())
	if len(out) > MaxEncoded {
		return "", fmt.Errorf("%w: %d characters, the limit is %d; share as a .mortar file instead", ErrTooLarge, len(out), MaxEncoded)
	}
	return out, nil
}

// roundKB keeps two significant digits, which is all the page shows and compresses far better than exact sizes.
func roundKB(kb int64) int64 {
	scale := int64(1)
	for kb >= 100*scale {
		scale *= 10
	}
	return (kb + scale/2) / scale * scale
}

func withoutDetails(s Shared) Shared {
	out := Shared{Game: s.Game, SourceKeys: s.SourceKeys, Name: s.Name, Entries: make([]Ref, len(s.Entries)), GameVersion: s.GameVersion}
	for i, r := range s.Entries {
		out.Entries[i] = Ref{ModID: r.ModID, FileID: r.FileID, GitHub: r.GitHub, Package: r.Package, Version: r.Version}
	}
	return out
}

// refOf maps an enabled, non-bundled entry to its Ref, or says why it cannot be shared.
func refOf(e profile.Entry, fomod, notes bool) (Ref, string) {
	var r Ref
	var missing string
	switch e.Source.Kind {
	case profile.KindNexus:
		r = Ref{ModID: e.Source.ModID, FileID: e.Source.FileID}
		missing = "no Nexus file recorded"
	case profile.KindGitHub:
		r = Ref{GitHub: e.Source.Repo + "@" + e.Source.Tag + "/" + e.Source.Asset}
		missing = "no GitHub release asset recorded"
	case profile.KindThunderstore:
		r = Ref{Package: e.Source.Name, Version: e.Source.Version}
		missing = "no Thunderstore package recorded"
	case profile.KindLocal:
		return Ref{}, "local archive"
	default:
		return Ref{}, "unknown source"
	}
	r.Disabled = slices.Clone(e.Disabled)
	if e.IsOverlay() {
		if e.Source.Kind != profile.KindNexus {
			return Ref{}, "optional file from a source a link cannot carry"
		}
		r.Overlay = &Overlay{From: e.OverlayFrom, To: e.OverlayTo, Off: e.OverlayOff}
	}
	if fomod {
		r.Fomod = cloneFomod(e.Fomod)
	}
	if notes {
		r.Note = e.Note
		r.Tags = slices.Clone(e.Tags)
	}
	if !r.valid() {
		return Ref{}, missing
	}
	return r, ""
}

// Enabled reports whether any mod of the entry is switched on; an entry with no recorded mods counts as on.
func Enabled(e profile.Entry) bool {
	if e.IsOverlay() {
		return !e.OverlayOff
	}
	if len(e.Mods) == 0 {
		return true
	}
	for _, m := range e.Mods {
		if e.Enabled(m.ID) {
			return true
		}
	}
	return false
}

// Collect splits a profile into what a link can carry and the enabled entries it cannot; off holds the keys of
// the switched-off entries. Bundled entries are in none of them.
func Collect(p profile.Profile, include ...Include) (s Shared, left []LeftOut, off []string) {
	inc := DefaultInclude()
	if len(include) > 0 {
		inc = include[0]
	}
	s = Shared{Name: p.Name, Entries: []Ref{}}
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		if !Enabled(e) && !inc.DisabledMods {
			off = append(off, e.Key)
			continue
		}
		r, why := refOf(e, inc.FomodChoices, inc.Notes)
		if why != "" {
			left = append(left, LeftOut{Key: e.Key, Reason: why})
			continue
		}
		r.key = e.Key
		s.Entries = append(s.Entries, r)
	}
	return s, left, off
}

// SourceKeys is each of the game's catalog sources that names the game, keyed by source id.
func SourceKeys(gameID string) map[string]string {
	out := map[string]string{}
	for _, g := range game.Catalog() {
		if g.ID != gameID {
			continue
		}
		for _, src := range g.Sources {
			if src.Key != "" {
				out[src.ID] = src.Key
			}
		}
	}
	return out
}

// linkPrefix matches a web or app link up to its payload; the game id is group 1 or 2.
var linkPrefix = regexp.MustCompile(`^(?:https://mortar\.rethunk\.tech/([a-z0-9-]+)/p#|mortar://([a-z0-9-]+)/p/)`)

// Encode turns a profile of the game into its share links. facts are what the share page shows beside the mods (the
// zero value shows nothing more).
func Encode(gameID string, p profile.Profile, facts profile.ShareFacts, include ...Include) (Result, error) {
	s, left, _ := Collect(p, include...)
	s.Game, s.SourceKeys = gameID, SourceKeys(gameID)
	plain := s
	s.GameVersion = facts.GameVersion
	s.Entries = slices.Clone(s.Entries)
	for i, r := range s.Entries {
		s.Entries[i].SizeKB, s.Entries[i].MinGame = roundKB(facts.SizeKB[r.key]), facts.MinGame[r.key]
	}
	// What the page shows goes before what the install uses: a link too large drops the sizes and minimum versions
	// first, then the notes, choices and disabled mods.
	payload, err := s.payload()
	for _, fallback := range []Shared{plain, withoutDetails(plain)} {
		if !errors.Is(err, ErrTooLarge) {
			break
		}
		if payload, err = fallback.payload(); err == nil {
			s = fallback
		}
	}
	if err != nil {
		return Result{}, err
	}
	return Result{Shared: s, Payload: payload, Web: "https://mortar.rethunk.tech/" + gameID + "/p#" + payload, App: "mortar://" + gameID + "/p/" + payload, LeftOut: left}, nil
}

// Parse accepts the web link, the mortar:// link, or a bare payload, and returns what it names. The game id is
// returned unchecked: the caller decides which games it knows.
func Parse(text string) (Shared, error) {
	text = strings.TrimSpace(text)
	payload, linkGame := text, ""
	if strings.Contains(text, ":") {
		m := linkPrefix.FindStringSubmatch(text)
		if m == nil {
			return Shared{}, ErrNotLink
		}
		linkGame = m[1] + m[2]
		payload = strings.TrimSuffix(text[len(m[0]):], "/")
	}
	s, err := decode(payload)
	if err == nil && linkGame != "" && linkGame != s.Game {
		return Shared{}, fmt.Errorf("%w: link and payload name different games", ErrMalformed)
	}
	return s, err
}

func decode(payload string) (Shared, error) {
	if payload == "" {
		return Shared{}, ErrNotLink
	}
	if len(payload) > MaxEncoded {
		return Shared{}, ErrTooLarge
	}
	packed, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return Shared{}, ErrNotLink
	}
	raw, err := io.ReadAll(io.LimitReader(brotli.NewReader(bytes.NewReader(packed)), MaxDecoded+1))
	if err != nil {
		return Shared{}, fmt.Errorf("%w: not compressed data", ErrMalformed)
	}
	if len(raw) > MaxDecoded {
		return Shared{}, ErrTooLarge
	}
	doc, err := versioned(raw, 5)
	if err != nil {
		return Shared{}, err
	}
	var s Shared
	if err := json.Unmarshal(doc[1], &s.Name); err != nil {
		return Shared{}, fmt.Errorf("%w: bad profile name", ErrMalformed)
	}
	if json.Unmarshal(doc[2], &s.Game) != nil || json.Unmarshal(doc[3], &s.SourceKeys) != nil {
		return Shared{}, fmt.Errorf("%w: bad game or source keys", ErrMalformed)
	}
	if s.Entries, err = parseEntries(doc[4]); err != nil {
		return Shared{}, err
	}
	if len(doc) > 5 && json.Unmarshal(doc[5], &s.GameVersion) != nil {
		return Shared{}, fmt.Errorf("%w: bad game version", ErrMalformed)
	}
	if err := checkShared(s); err != nil {
		return Shared{}, err
	}
	return s, nil
}
