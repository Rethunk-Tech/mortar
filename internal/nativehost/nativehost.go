// Package nativehost is Mortar's side of the browser extension's native messaging: the browser starts Mortar with
// the calling extension's origin, writes length-prefixed JSON messages to its stdin and reads replies from stdout.
package nativehost

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

const (
	// Name is the native messaging host name the extension connects to.
	Name = "tech.rethunk.mortar"
	// ChromeOrigin is the unpacked extension's origin, fixed by the key in the extension's manifest.json
	// (Rethunk-Tech/mortar-browser-extension).
	ChromeOrigin = "chrome-extension://ifegpceemlkelinckndpmnaeepfcoplj/"
	// FirefoxID is the extension's gecko id.
	FirefoxID = "mortar@rethunk.tech"
	// maxMessage bounds what Mortar reads; a link is a few hundred bytes.
	maxMessage = 64 * 1024
	// Protocol is the native-messaging protocol Mortar speaks, sent in every reply; the extension checks it against
	// its own range. MinProtocol and MaxProtocol are the extension protocols Mortar answers; a request that sends
	// none counts as protocol 0.
	Protocol    = 2
	MinProtocol = 2
	MaxProtocol = 2
)

// Protocol mismatches Mortar reports on a refused message and records for Settings.
const (
	ExtensionTooOld = "extensionTooOld"
	ExtensionTooNew = "extensionTooNew"
)

// protocolMismatch is ExtensionTooOld or ExtensionTooNew when Mortar does not speak the extension's protocol, else "".
func protocolMismatch(sent *int) string {
	protocol := 0
	if sent != nil {
		protocol = *sent
	}
	switch {
	case protocol < MinProtocol:
		return ExtensionTooOld
	case protocol > MaxProtocol:
		return ExtensionTooNew
	}
	return ""
}

// storeChromeIDs are the ids the Chrome Web Store and Edge Add-ons assign the extension, which differ from the unpacked
// id; each is allowed to reach the host.
var storeChromeIDs = []string{
	"hboeppdoecbglcmfojappgehkbecdfbi", // Chrome Web Store
}

// ChromeOrigins lists every extension origin the host manifest allows: the unpacked build plus the store builds.
func ChromeOrigins() []string {
	origins := []string{ChromeOrigin}
	for _, id := range storeChromeIDs {
		origins = append(origins, "chrome-extension://"+id+"/")
	}
	return origins
}

// Invoked reports whether args (without the program name) are a browser starting Mortar as a native host: Chromium
// passes the caller's origin first, Firefox the manifest path and then the extension id.
func Invoked(args []string) bool {
	if len(args) > 0 && strings.HasPrefix(args[0], "chrome-extension://") {
		return true
	}
	return len(args) > 1 && args[1] == FirefoxID
}

type request struct {
	Protocol *int   `json:"protocol"`
	Type     string `json:"type"`
	Link     string `json:"link"`
	// Source and SourceGameKey name the page's game as its source does (nexus, "stardewvalley").
	Source        string `json:"source"`
	SourceGameKey string `json:"sourceGameKey"`
	ModID         int    `json:"modId"`
}

// gameKey is the page's Nexus domain, the key the handlers resolve to a Mortar game; empty for a source Mortar does
// not read pages of.
func (r request) gameKey() string {
	if r.Source != "nexus" {
		return ""
	}
	return r.SourceGameKey
}

// hostGame is a game Mortar manages, with each source's name for it.
type hostGame struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Sources map[string]string `json:"sources"`
}

type reply struct {
	Protocol      int               `json:"protocol"`
	ProtocolError string            `json:"protocolError,omitempty"`
	OK            bool              `json:"ok,omitempty"`
	Error         string            `json:"error,omitempty"`
	Connected     bool              `json:"connected"`
	ModIDs        *[]int            `json:"modIds,omitempty"`
	BrokenIDs     []int             `json:"brokenIds,omitempty"`
	Open          *modInProfile     `json:"open,omitempty"`
	Others        []modInProfile    `json:"others,omitempty"`
	Problems      []modProblem      `json:"problems,omitempty"`
	Updates       *[]modUpdate      `json:"updates,omitempty"`
	Requirements  []requirementItem `json:"requirements,omitempty"`
	Profile       string            `json:"profile,omitempty"`
	State         string            `json:"state,omitempty"`
	UpdateIDs     []int             `json:"updateIds,omitempty"`
	// Games are the games Mortar manages, so the extension draws its UI only on those and maps a page's source key
	// to the Mortar game id.
	Games []hostGame `json:"games,omitempty"`
	// Accent is Mortar's accent colour, read on every reply so the extension follows a change in the app.
	Accent string `json:"accent,omitempty"`
}

type modUpdate struct {
	ModID     int    `json:"modId"`
	Name      string `json:"name"`
	Installed string `json:"installed"`
	Latest    string `json:"latest"`
}

type modProblem struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type modInProfile struct {
	Profile         string   `json:"profile"`
	Version         *string  `json:"version"`
	FileID          int      `json:"fileId,omitempty"`
	UpdateAvailable bool     `json:"updateAvailable,omitempty"`
	RequiredBy      []string `json:"requiredBy,omitempty"`
	RequiredByNames []string `json:"requiredByNames,omitempty"`
	Pinned          bool     `json:"pinned,omitempty"`
	PinReason       string   `json:"pinReason,omitempty"`
	SkipVersion     string   `json:"skipVersion,omitempty"`
	SkipSources     []string `json:"skipSources,omitempty"`
	// PageVersion is the version Nexus lists for the mod, from Mortar's cached details.
	PageVersion string `json:"pageVersion,omitempty"`
}

type diskMod struct {
	Name           string   `json:"name"`
	ID             mod.ID   `json:"id"`
	Version        string   `json:"version"`
	Needs          []mod.ID `json:"needs"`
	Optional       []mod.ID `json:"optional"`
	ContentPackFor mod.ID   `json:"contentPackFor"`
}

type diskEntry struct {
	Pinned      bool     `json:"pinned"`
	PinReason   string   `json:"pinReason,omitempty"`
	SkipVersion string   `json:"skipVersion"`
	SkipSources []string `json:"skipSources"`
	Disabled    []mod.ID `json:"disabled"`
	Source      struct {
		Kind   string `json:"kind"`
		ModID  int    `json:"modId"`
		FileID int    `json:"fileId"`
	} `json:"source"`
	Mods []diskMod `json:"mods"`
}

func requiresID(m diskMod, target mod.ID) bool {
	if mod.Equal(m.ContentPackFor, target) {
		return true
	}
	opt := map[string]bool{}
	for _, id := range m.Optional {
		opt[id.Fold()] = true
	}
	for _, id := range m.Needs {
		if mod.Equal(id, target) && !opt[id.Fold()] {
			return true
		}
	}
	return false
}

type requiredByRef struct {
	id   mod.ID
	name string
}

func requiredByMods(entries []diskEntry, targets []mod.ID) ([]string, []string) {
	seen := map[string]bool{}
	offOf := func(disabled []mod.ID) map[string]bool {
		m := map[string]bool{}
		for _, id := range disabled {
			m[id.Fold()] = true
		}
		return m
	}
	var refs []requiredByRef
	for _, e := range entries {
		off := offOf(e.Disabled)
		for _, m := range e.Mods {
			id := m.ID.Fold()
			if id == "" || off[id] || seen[id] {
				continue
			}
			if slices.ContainsFunc(targets, func(t mod.ID) bool { return mod.Equal(t, m.ID) }) {
				continue
			}
			for _, t := range targets {
				if requiresID(m, t) {
					seen[id] = true
					name := m.Name
					if name == "" {
						name = m.ID.Local()
					}
					refs = append(refs, requiredByRef{id: m.ID, name: name})
					break
				}
			}
		}
	}
	slices.SortFunc(refs, func(a, b requiredByRef) int {
		return strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
	})
	ids := make([]string, len(refs))
	names := make([]string, len(refs))
	for i, ref := range refs {
		ids[i] = ref.id.Local()
		names[i] = ref.name
	}
	return ids, names
}

// Host states a data reply carries, so the extension can tell a missing Mortar from one that is closed, one with no
// profile open for the page's game, and a connection the user turned off in Mortar's settings.
const (
	stateOff        = "off"
	stateNotRunning = "notRunning"
	stateNoProfile  = "noProfile"
	stateReady      = "ready"
)

type handlers struct {
	contact      func(mismatch string)
	open         func(link string) error
	installed    func(game string) []int
	im           func(game string, modID int) (modInProfile, []modInProfile)
	state        func(game string) (state, profile string)
	updates      func(game string) (string, []modUpdate)
	broken       func(game string) []int
	problems     func(game string, modID int) []modProblem
	requirements func(game string, modID int) []requirementItem
}

// Serve answers messages from r until it closes, handing each message's link to open.
func Serve(r io.Reader, w io.Writer, open func(link string) error) error {
	return ServeFrom(nil, r, w, open)
}

// ServeFrom is Serve for a host the browser started with args (see Invoked); each message records that the
// browser is in contact.
func ServeFrom(args []string, r io.Reader, w io.Writer, open func(link string) error) error {
	browser := browserName(args)
	return serveHandlers(r, w, handlers{
		contact: func(mismatch string) { recordContact(browser, mismatch, time.Now()) },
		open:    open, installed: activeNexusModIDs, im: nexusModProfiles, state: activeNexusState,
		updates: activeNexusUpdates, broken: brokenNexusModIDs, problems: nexusModProblems,
		requirements: nexusPageRequirements,
	})
}

// answer is the reply to one data request; a link is handled by the caller.
func (h handlers) answer(req request) reply {
	st, name := stateReady, ""
	if h.state != nil {
		st, name = h.state(req.gameKey())
	}
	var rep reply
	switch req.Type {
	case "installed":
		ids := []int{}
		if st != stateOff {
			if h.installed != nil {
				if found := h.installed(req.gameKey()); found != nil {
					ids = found
				}
			}
			if h.broken != nil {
				rep.BrokenIDs = h.broken(req.gameKey())
			}
			if h.updates != nil && st == stateReady {
				_, rows := h.updates(req.gameKey())
				for _, row := range rows {
					rep.UpdateIDs = append(rep.UpdateIDs, row.ModID)
				}
			}
		}
		rep.ModIDs = &ids
	case "mod":
		if st == stateOff {
			break
		}
		openProfile, others := h.im(req.gameKey(), req.ModID)
		rep.Open, rep.Others = &openProfile, others
		if st == stateReady {
			if h.problems != nil {
				rep.Problems = h.problems(req.gameKey(), req.ModID)
			}
			if h.requirements != nil {
				rep.Requirements = h.requirements(req.gameKey(), req.ModID)
			}
		}
	case "updates":
		rows := []modUpdate{}
		if st != stateOff && h.updates != nil {
			name, rows = h.updates(req.gameKey())
		}
		if rows == nil {
			rows = []modUpdate{}
		}
		slices.SortFunc(rows, func(a, b modUpdate) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})
		rep.Updates = &rows
	default:
		return reply{Error: fmt.Sprintf("unknown request type %q", req.Type)}
	}
	rep.State, rep.Connected, rep.Profile = st, st == stateReady, name
	rep.Games = hostGames()
	return rep
}

func serveHandlers(r io.Reader, w io.Writer, h handlers) error {
	for {
		var n uint32
		if err := binary.Read(r, binary.NativeEndian, &n); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if n > maxMessage {
			return fmt.Errorf("message of %d bytes is too large", n)
		}
		var req request
		if err := json.NewDecoder(io.LimitReader(r, int64(n))).Decode(&req); err != nil {
			return err
		}
		mismatch := protocolMismatch(req.Protocol)
		if h.contact != nil {
			h.contact(mismatch)
		}
		var rep reply
		switch {
		case mismatch != "":
			rep = reply{Error: "Mortar does not speak this browser extension's protocol", ProtocolError: mismatch}
		case req.Type == "":
			rep = reply{OK: true}
			if err := h.open(req.Link); err != nil {
				rep = reply{Error: err.Error()}
			}
		default:
			rep = h.answer(req)
		}
		rep.Protocol = Protocol
		rep.Accent = accentColor()
		out, err := json.Marshal(rep)
		if err != nil {
			return err
		}
		n32, err := frameLen(len(out))
		if err != nil {
			return err
		}
		if err := binary.Write(w, binary.NativeEndian, n32); err != nil {
			return err
		}
		if _, err := w.Write(out); err != nil {
			return err
		}
	}
}

func nexusModProblems(domain string, modID int) []modProblem {
	if modID < 1 {
		return []modProblem{}
	}
	info, ok := components.BundledGameByNexusDomain(domain)
	if !ok {
		return []modProblem{}
	}
	dataDir, err := datadir.Dir()
	if err != nil || !controlwire.Running(dataDir) {
		return []modProblem{}
	}
	store, err := settings.Open()
	if err != nil {
		return []modProblem{}
	}
	profileID := store.Get().LastProfile[info.ID]
	if profileID == "" || filepath.Base(profileID) != profileID {
		return []modProblem{}
	}
	var rows []modProblem
	if err := controlwire.CallDir(dataDir, "modProblems", map[string]any{
		"game": info.ID, "profile": profileID, "modId": modID,
	}, &rows, time.Second); err != nil {
		return []modProblem{}
	}
	return rows
}

// brokenNexusModIDs lists the Nexus pages SMAPI's compatibility list marks broken for the game version last played,
// from the copy Mortar cached for its Problems check.
func brokenNexusModIDs(domain string) []int {
	info, ok := components.BundledGameByNexusDomain(domain)
	if !ok || !info.HasMetadata("smapi-compat") {
		return nil
	}
	idx, ok := meta.Peek[meta.CompatIndex](&meta.Client{}, meta.CompatCacheFile)
	if !ok {
		return nil
	}
	gameVersion := ""
	if store, err := settings.Open(); err == nil {
		gameVersion = store.Get().LastPlayed[info.ID].GameVersion
	}
	ids := []int{}
	for id, e := range idx.ByNexus {
		if e.BrokenOn(gameVersion) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}

func activeNexusModIDs(domain string) []int {
	ids := []int{}
	info, ok := components.BundledGameByNexusDomain(domain)
	if !ok {
		return ids
	}
	dataDir, err := datadir.Dir()
	if err != nil || !controlwire.Running(dataDir) {
		return ids
	}
	store, err := settings.Open()
	if err != nil {
		return ids
	}
	// The profile last open for the page's game, even while Mortar shows another game.
	profileID := store.Get().LastProfile[info.ID]
	if profileID == "" || filepath.Base(profileID) != profileID {
		return ids
	}
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		return ids
	}
	data, err := root.ReadFile(filepath.Join("profiles", info.ID, profileID, "profile.json"))
	_ = root.Close()
	if err != nil {
		return ids
	}
	var profile struct {
		Entries []struct {
			Source struct {
				Kind  string `json:"kind"`
				Name  string `json:"name"`
				ModID int    `json:"modId"`
			} `json:"source"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return ids
	}
	seen := make(map[int]struct{}, len(profile.Entries))
	add := func(id int) {
		if _, exists := seen[id]; id > 0 && !exists {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	for _, entry := range profile.Entries {
		switch entry.Source.Kind {
		case "nexus":
			add(entry.Source.ModID)
		case "smapi":
			add(info.LoaderNexusModID())
		case "local":
			add(nexusArchiveModID(entry.Source.Name))
		}
	}
	return ids
}

// nexusArchiveName matches the file name Nexus gives a download, "<name>-<mod id>-<version parts>-<unix time>.zip",
// which a manually downloaded archive keeps when it is added to a profile.
var nexusArchiveName = regexp.MustCompile(`-(\d+)(?:-[0-9a-zA-Z]+)*-\d{10}\.(?:zip|7z|rar)$`)

// nexusArchiveModID is the Nexus mod id in a Nexus download's file name, or 0.
func nexusArchiveModID(name string) int {
	m := nexusArchiveName.FindStringSubmatch(name)
	if m == nil {
		return 0
	}
	id, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return id
}

func activeNexusUpdates(domain string) (string, []modUpdate) {
	rows := []modUpdate{}
	info, ok := components.BundledGameByNexusDomain(domain)
	if !ok {
		return "", rows
	}
	dataDir, err := datadir.Dir()
	if err != nil || !controlwire.Running(dataDir) {
		return "", rows
	}
	store, err := settings.Open()
	if err != nil {
		return "", rows
	}
	profileID := store.Get().LastProfile[info.ID]
	if profileID == "" || filepath.Base(profileID) != profileID {
		return "", rows
	}
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		return "", rows
	}
	defer func() { _ = root.Close() }()
	data, err := root.ReadFile(filepath.Join("profiles", info.ID, profileID, "profile.json"))
	if err != nil {
		return "", rows
	}
	var profile struct {
		Name    string      `json:"name"`
		Entries []diskEntry `json:"entries"`
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return "", rows
	}
	seen := map[int]struct{}{}
	for _, entry := range profile.Entries {
		if entry.Source.Kind != "nexus" || entry.Source.ModID < 1 {
			continue
		}
		if _, dup := seen[entry.Source.ModID]; dup {
			continue
		}
		var version *string
		installed := ""
		name := ""
		if len(entry.Mods) > 0 {
			name = entry.Mods[0].Name
			if name == "" {
				name = entry.Mods[0].ID.Local()
			}
			if entry.Mods[0].Version != "" {
				v := entry.Mods[0].Version
				version = &v
				installed = v
			}
		}
		pageVer, files := cachedNexusDetails(root, info.NexusDomain(), entry.Source.ModID)
		if !nexusUpdateAvailable(version, entry.Source.FileID, pageVer, files) {
			continue
		}
		seen[entry.Source.ModID] = struct{}{}
		rows = append(rows, modUpdate{
			ModID:     entry.Source.ModID,
			Name:      name,
			Installed: installed,
			Latest:    pageVer,
		})
	}
	slices.SortFunc(rows, func(a, b modUpdate) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return profile.Name, rows
}

// hostGames are the games Mortar manages that a source names.
func hostGames() []hostGame {
	m, err := components.BundledManifest()
	if err != nil {
		return nil
	}
	var games []hostGame
	for _, g := range m.Games {
		keys := map[string]string{}
		for _, src := range g.Sources {
			if src.Key != "" {
				keys[src.ID] = src.Key
			}
		}
		if len(keys) > 0 {
			games = append(games, hostGame{ID: g.ID, Name: g.Name, Sources: keys})
		}
	}
	return games
}

// activeNexusState is how far the page's game is from showing Mortar data, and the open profile's name when it is
// ready. Off wins over everything: the user turned the connection off in Mortar.
func activeNexusState(domain string) (string, string) {
	info, ok := components.BundledGameByNexusDomain(domain)
	if !ok {
		return stateNoProfile, ""
	}
	dataDir, err := datadir.Dir()
	if err != nil {
		return stateNotRunning, ""
	}
	store, err := settings.Open()
	if err != nil {
		return stateNotRunning, ""
	}
	cur := store.Get()
	if conn, err := cur.Lookup("extensionConnection"); err == nil && conn == settings.ExtensionOff {
		return stateOff, ""
	}
	if !controlwire.Running(dataDir) {
		return stateNotRunning, ""
	}
	profileID := cur.LastProfile[info.ID]
	if profileID == "" || filepath.Base(profileID) != profileID {
		return stateNoProfile, ""
	}
	return stateReady, profileName(dataDir, info.ID, profileID)
}

func profileName(dataDir, gameID, profileID string) string {
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		return ""
	}
	defer func() { _ = root.Close() }()
	data, err := root.ReadFile(filepath.Join("profiles", gameID, profileID, "profile.json"))
	if err != nil {
		return ""
	}
	var p struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(data, &p)
	return p.Name
}

func nexusModProfiles(domain string, modID int) (modInProfile, []modInProfile) {
	var openProfile modInProfile
	var others []modInProfile
	if modID < 1 {
		return openProfile, nil
	}
	info, ok := components.BundledGameByNexusDomain(domain)
	if !ok {
		return openProfile, nil
	}
	dataDir, err := datadir.Dir()
	if err != nil || !controlwire.Running(dataDir) {
		return openProfile, nil
	}
	store, err := settings.Open()
	if err != nil {
		return openProfile, nil
	}
	openID := store.Get().LastProfile[info.ID]
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		return openProfile, nil
	}
	defer func() { _ = root.Close() }()
	pageVer, files := cachedNexusDetails(root, info.NexusDomain(), modID)
	profiles, err := root.Open(filepath.Join("profiles", info.ID))
	if err != nil {
		return openProfile, nil
	}
	entries, err := profiles.ReadDir(-1)
	_ = profiles.Close()
	if err != nil {
		return openProfile, nil
	}
	for _, dir := range entries {
		if !dir.IsDir() || strings.HasPrefix(dir.Name(), ".") {
			continue
		}
		var profile struct {
			Name    string      `json:"name"`
			Hidden  bool        `json:"hidden"`
			Entries []diskEntry `json:"entries"`
		}
		data, err := root.ReadFile(filepath.Join("profiles", info.ID, dir.Name(), "profile.json"))
		if err != nil || json.Unmarshal(data, &profile) != nil || profile.Hidden {
			continue
		}
		var version *string
		var fileID int
		var targets []mod.ID
		var pinned bool
		var pinReason string
		var skipVersion string
		var skipSources []string
		for _, entry := range profile.Entries {
			if entry.Source.Kind != "nexus" || entry.Source.ModID != modID {
				continue
			}
			if len(entry.Mods) > 0 && entry.Mods[0].Version != "" {
				v := entry.Mods[0].Version
				version = &v
			}
			fileID = entry.Source.FileID
			pinned = entry.Pinned
			pinReason = entry.PinReason
			skipVersion = entry.SkipVersion
			skipSources = append([]string{}, entry.SkipSources...)
			for _, m := range entry.Mods {
				if m.ID != "" {
					targets = append(targets, m.ID)
				}
			}
			break
		}
		found := modInProfile{Profile: profile.Name, Version: version, FileID: fileID}
		if version != nil || fileID > 0 {
			found.UpdateAvailable = nexusUpdateAvailable(version, fileID, pageVer, files)
		}
		if dir.Name() == openID {
			found.Pinned = pinned
			found.PinReason = pinReason
			found.SkipVersion = skipVersion
			found.SkipSources = skipSources
			found.RequiredBy, found.RequiredByNames = requiredByMods(profile.Entries, targets)
			found.PageVersion = pageVer
			openProfile = found
		} else {
			others = append(others, found)
		}
	}
	return openProfile, others
}

type cachedNexusFile struct {
	FileID     int    `json:"fileId"`
	Version    string `json:"version"`
	ReplacedBy int    `json:"replacedBy"`
}

func cachedNexusDetails(root *os.Root, domain string, modID int) (string, []cachedNexusFile) {
	c := &meta.Client{CacheDir: filepath.Join(root.Name(), "cache")}
	d, ok := meta.Peek[nexussvc.Details](c, nexussvc.DetailsName(domain, modID))
	if !ok {
		return "", nil
	}
	files := make([]cachedNexusFile, 0, len(d.Files))
	for _, f := range d.Files {
		files = append(files, cachedNexusFile{FileID: f.FileID, Version: f.Version, ReplacedBy: f.ReplacedBy})
	}
	return d.Page.Version, files
}

func nexusUpdateAvailable(version *string, fileID int, pageVer string, files []cachedNexusFile) bool {
	if version != nil && meta.Newer(pageVer, *version) {
		return true
	}
	if fileID < 1 {
		return false
	}
	byID := make(map[int]cachedNexusFile, len(files))
	for _, f := range files {
		byID[f.FileID] = f
	}
	cur := fileID
	seen := map[int]bool{}
	for {
		f, ok := byID[cur]
		if !ok || f.ReplacedBy < 1 || seen[cur] {
			return cur != fileID
		}
		seen[cur] = true
		cur = f.ReplacedBy
	}
}

// frameLen is a message's length as the uint32 prefix native messaging uses.
func frameLen(n int) (uint32, error) {
	if n < 0 || uint64(n) > math.MaxUint32 {
		return 0, fmt.Errorf("message of %d bytes cannot be framed", n)
	}
	return uint32(n), nil
}

// Manifest is the host manifest a browser reads to find Mortar; firefox picks that browser's allow-list field.
func Manifest(exe string, firefox bool) ([]byte, error) {
	m := map[string]any{
		"name":        Name,
		"description": "Mortar mod manager",
		"path":        exe,
		"type":        "stdio",
	}
	if firefox {
		m["allowed_extensions"] = []string{FirefoxID}
	} else {
		m["allowed_origins"] = ChromeOrigins()
	}
	return json.MarshalIndent(m, "", "  ")
}

// accentColor reads only the accent from Mortar's settings file; the native host never writes Mortar's data, so it
// does not go through settings.Open, which moves a corrupt file aside.
func accentColor() string {
	var s struct {
		Global struct {
			Accent string `json:"accent"`
		} `json:"global"`
	}
	if dir, err := datadir.Dir(); err == nil {
		if b, err := fsx.ReadFile(filepath.Join(dir, settings.FileName)); err == nil {
			_ = json.Unmarshal(b, &s)
		}
	}
	return settings.AccentColor(s.Global.Accent)
}
