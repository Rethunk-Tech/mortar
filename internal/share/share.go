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
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/andybalholm/brotli"

	"github.com/Rethunk-AI/mortar/internal/profile"
)

const (
	// FormatVersion is the payload's leading number.
	FormatVersion = 1

	MaxEncoded    = 8 << 10
	MaxDecoded    = 64 << 10
	MaxEntries    = 1000
	MaxNameLength = 60

	maxID     = 1<<31 - 1
	webPrefix = "https://mortar.rethunk.tech/stardew/p#"
	appPrefix = "mortar://stardew/p/"
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

// Ref names one file to install: a Nexus mod file, or a GitHub release asset as "<owner>/<repo>@<tag>/<asset>".
type Ref struct {
	ModID  int
	FileID int
	GitHub string
}

// Shared is what a link carries.
type Shared struct {
	Name    string
	Entries []Ref
}

// LeftOut is an enabled entry that cannot travel in a link.
type LeftOut struct {
	Key    string
	Reason string
}

// Result is an encoded profile: the payload, both link forms, and the entries left out.
type Result struct {
	Shared  Shared
	Payload string
	Web     string
	App     string
	LeftOut []LeftOut
}

var githubRef = regexp.MustCompile(`^[A-Za-z0-9-]{1,39}/[A-Za-z0-9._-]{1,100}@[A-Za-z0-9._+-]{1,100}/[A-Za-z0-9._+~()-]{1,200}$`)

func validName(name string) bool {
	n := utf8.RuneCountInString(name)
	return n > 0 && n <= MaxNameLength && strings.TrimSpace(name) == name &&
		!strings.ContainsFunc(name, unicode.IsControl)
}

func (r Ref) valid() bool {
	if r.GitHub != "" {
		return r.ModID == 0 && r.FileID == 0 && githubRef.MatchString(r.GitHub)
	}
	return r.ModID > 0 && r.ModID <= maxID && r.FileID > 0 && r.FileID <= maxID
}

// MarshalJSON writes a GitHub ref as its string and a Nexus ref as [mod id, file id].
func (r Ref) MarshalJSON() ([]byte, error) {
	if r.GitHub != "" {
		return json.Marshal(r.GitHub)
	}
	return json.Marshal([2]int{r.ModID, r.FileID})
}

func parseRef(raw json.RawMessage) (Ref, error) {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return Ref{GitHub: s}, nil
	}
	var ids []int
	if err := json.Unmarshal(raw, &ids); err != nil || len(ids) != 2 {
		return Ref{}, fmt.Errorf("%w: bad entry", ErrMalformed)
	}
	return Ref{ModID: ids[0], FileID: ids[1]}, nil
}

func checkShared(s Shared) error {
	if !validName(s.Name) {
		return fmt.Errorf("%w: bad profile name", ErrMalformed)
	}
	if len(s.Entries) > MaxEntries {
		return fmt.Errorf("%w: more than %d entries", ErrMalformed, MaxEntries)
	}
	for _, r := range s.Entries {
		if !r.valid() {
			return fmt.Errorf("%w: bad entry", ErrMalformed)
		}
	}
	return nil
}

// versioned splits a payload document into its version and the rest, refusing a newer version before the rest
// is looked at, since its shape may differ.
func versioned(raw []byte, parts int) ([]json.RawMessage, error) {
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
	if len(doc) != parts {
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
	raw := fmt.Appendf(nil, "[%d,%s,%s]", FormatVersion, name, entries)
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

// refOf maps an enabled, non-bundled entry to its Ref, or says why it cannot be shared.
func refOf(e profile.Entry) (Ref, string) {
	switch e.Source.Kind {
	case profile.KindNexus:
		r := Ref{ModID: e.Source.ModID, FileID: e.Source.FileID}
		if !r.valid() {
			return Ref{}, "no Nexus file recorded"
		}
		return r, ""
	case profile.KindGitHub:
		r := Ref{GitHub: e.Source.Repo + "@" + e.Source.Tag + "/" + e.Source.Asset}
		if !r.valid() {
			return Ref{}, "no GitHub release asset recorded"
		}
		return r, ""
	case profile.KindLocal:
		return Ref{}, "local archive"
	default:
		return Ref{}, "unknown source"
	}
}

func bundled(e profile.Entry) bool {
	return e.Source.Kind == profile.SourceSMAPI || e.Source.Kind == profile.SourceMortar
}

// enabled reports whether any mod of the entry is switched on; an entry with no recorded mods counts as on.
func enabled(e profile.Entry) bool {
	if len(e.Mods) == 0 {
		return true
	}
	for _, m := range e.Mods {
		if !containsFold(e.Disabled, m.UniqueID) {
			return true
		}
	}
	return false
}

func containsFold(ids []string, id string) bool {
	for _, x := range ids {
		if strings.EqualFold(x, id) {
			return true
		}
	}
	return false
}

// collect splits a profile into what a link can carry and the enabled entries it cannot.
// Bundled and switched-off entries are in neither.
func collect(p profile.Profile) (Shared, []LeftOut) {
	s := Shared{Name: p.Name, Entries: []Ref{}}
	var left []LeftOut
	for _, e := range p.Entries {
		if bundled(e) || !enabled(e) {
			continue
		}
		r, why := refOf(e)
		if why != "" {
			left = append(left, LeftOut{Key: e.Key, Reason: why})
			continue
		}
		s.Entries = append(s.Entries, r)
	}
	return s, left
}

// Encode turns a profile into its share links.
func Encode(p profile.Profile) (Result, error) {
	s, left := collect(p)
	payload, err := s.payload()
	if err != nil {
		return Result{}, err
	}
	return Result{Shared: s, Payload: payload, Web: webPrefix + payload, App: appPrefix + payload, LeftOut: left}, nil
}

// Parse accepts the web link, the mortar:// link, or a bare payload, and returns what it names.
func Parse(text string) (Shared, error) {
	text = strings.TrimSpace(text)
	payload := text
	if strings.Contains(text, ":") {
		if p, ok := strings.CutPrefix(text, webPrefix); ok {
			payload = p
		} else if p, ok := strings.CutPrefix(text, appPrefix); ok {
			payload = strings.TrimSuffix(p, "/")
		} else {
			return Shared{}, ErrNotLink
		}
	}
	return decode(payload)
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
	doc, err := versioned(raw, 3)
	if err != nil {
		return Shared{}, err
	}
	var s Shared
	if err := json.Unmarshal(doc[1], &s.Name); err != nil {
		return Shared{}, fmt.Errorf("%w: bad profile name", ErrMalformed)
	}
	if s.Entries, err = parseEntries(doc[2]); err != nil {
		return Shared{}, err
	}
	if err := checkShared(s); err != nil {
		return Shared{}, err
	}
	return s, nil
}
