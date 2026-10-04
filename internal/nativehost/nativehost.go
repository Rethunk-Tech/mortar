// Package nativehost is Mortar's side of the browser extension's native messaging: the browser starts Mortar with
// the calling extension's origin, writes length-prefixed JSON messages to its stdin and reads replies from stdout.
package nativehost

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/components"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexussvc"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

const (
	// Name is the native messaging host name the extension connects to.
	Name = "tech.rethunk.mortar"
	// ChromeOrigin is the unpacked extension's origin, fixed by the key in browser-extension/manifest.json.
	ChromeOrigin = "chrome-extension://ifegpceemlkelinckndpmnaeepfcoplj/"
	// FirefoxID is the extension's gecko id.
	FirefoxID = "mortar@rethunk.tech"
	// maxMessage bounds what Mortar reads; a link is a few hundred bytes.
	maxMessage = 64 * 1024
)

// Invoked reports whether args (without the program name) are a browser starting Mortar as a native host: Chromium
// passes the caller's origin first, Firefox the manifest path and then the extension id.
func Invoked(args []string) bool {
	if len(args) > 0 && strings.HasPrefix(args[0], "chrome-extension://") {
		return true
	}
	return len(args) > 1 && args[1] == FirefoxID
}

type request struct {
	Type  string `json:"type"`
	Link  string `json:"link"`
	Game  string `json:"game"`
	ModID int    `json:"modId"`
}

type reply struct {
	OK           bool              `json:"ok,omitempty"`
	Error        string            `json:"error,omitempty"`
	Connected    bool              `json:"connected"`
	ModIDs       *[]int            `json:"modIds,omitempty"`
	BrokenIDs    []int             `json:"brokenIds,omitempty"`
	Open         *modInProfile     `json:"open,omitempty"`
	Others       []modInProfile    `json:"others,omitempty"`
	Problems     []modProblem      `json:"problems,omitempty"`
	Updates      *[]modUpdate      `json:"updates,omitempty"`
	Requirements []requirementItem `json:"requirements,omitempty"`
	Profile      string            `json:"profile,omitempty"`
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
}

type diskMod struct {
	Name           string   `json:"name"`
	UniqueID       string   `json:"uniqueId"`
	Version        string   `json:"version"`
	Needs          []string `json:"needs"`
	Optional       []string `json:"optional"`
	ContentPackFor string   `json:"contentPackFor"`
}

type diskEntry struct {
	Pinned      bool     `json:"pinned"`
	PinReason   string   `json:"pinReason,omitempty"`
	SkipVersion string   `json:"skipVersion"`
	SkipSources []string `json:"skipSources"`
	Disabled    []string `json:"disabled"`
	Source      struct {
		Kind   string `json:"kind"`
		ModID  int    `json:"modId"`
		FileID int    `json:"fileId"`
	} `json:"source"`
	Mods []diskMod `json:"mods"`
}

func foldID(id string) string { return strings.ToLower(strings.TrimSpace(id)) }

func requiresID(m diskMod, target string) bool {
	if strings.EqualFold(strings.TrimSpace(m.ContentPackFor), target) {
		return true
	}
	opt := map[string]bool{}
	for _, id := range m.Optional {
		opt[foldID(id)] = true
	}
	for _, id := range m.Needs {
		if strings.EqualFold(id, target) && !opt[foldID(id)] {
			return true
		}
	}
	return false
}

type requiredByRef struct {
	id   string
	name string
}

func requiredByMods(entries []diskEntry, targets []string) ([]string, []string) {
	seen := map[string]bool{}
	offOf := func(disabled []string) map[string]bool {
		m := map[string]bool{}
		for _, id := range disabled {
			m[foldID(id)] = true
		}
		return m
	}
	var refs []requiredByRef
	for _, e := range entries {
		off := offOf(e.Disabled)
		for _, m := range e.Mods {
			id := foldID(m.UniqueID)
			if id == "" || off[id] || seen[id] {
				continue
			}
			if slices.ContainsFunc(targets, func(t string) bool { return strings.EqualFold(t, m.UniqueID) }) {
				continue
			}
			for _, t := range targets {
				if requiresID(m, t) {
					seen[id] = true
					name := m.Name
					if name == "" {
						name = m.UniqueID
					}
					refs = append(refs, requiredByRef{id: m.UniqueID, name: name})
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
		ids[i] = ref.id
		names[i] = ref.name
	}
	return ids, names
}

// Serve answers messages from r until it closes, handing each message's link to open.
func Serve(r io.Reader, w io.Writer, open func(link string) error) error {
	return serveWithConnection(r, w, open, activeNexusModIDs, nexusModProfiles, activeNexusConnected, activeNexusUpdates, brokenNexusModIDs, nexusModProblems)
}

func serve(r io.Reader, w io.Writer, open func(link string) error, installed func(game string) []int, mod func(game string, modID int) (modInProfile, []modInProfile), problem ...func(game string, modID int) []modProblem) error {
	return serveWithConnection(r, w, open, installed, mod, nil, nil, nil, problem...)
}

func serveWithConnection(r io.Reader, w io.Writer, open func(link string) error, installed func(game string) []int, mod func(game string, modID int) (modInProfile, []modInProfile), connected func(string) bool, updates func(string) (string, []modUpdate), broken func(string) []int, problem ...func(game string, modID int) []modProblem) error {
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
		rep := reply{OK: true}
		switch req.Type {
		case "installed":
			ids := []int{}
			if installed != nil {
				if found := installed(req.Game); found != nil {
					ids = found
				}
			}
			isConnected := installed != nil
			if connected != nil {
				isConnected = connected(req.Game)
			}
			rep = reply{Connected: isConnected, ModIDs: &ids}
			if broken != nil {
				rep.BrokenIDs = broken(req.Game)
			}
		case "mod":
			openProfile, others := mod(req.Game, req.ModID)
			rep = reply{Open: &openProfile, Others: others}
		case "updates":
			name, rows := "", []modUpdate{}
			if updates != nil {
				name, rows = updates(req.Game)
			}
			if rows == nil {
				rows = []modUpdate{}
			}
			slices.SortFunc(rows, func(a, b modUpdate) int {
				return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
			})
			rep = reply{Updates: &rows, Profile: name}
		case "modProblems":
			var rows []modProblem
			if len(problem) > 0 && problem[0] != nil {
				rows = problem[0](req.Game, req.ModID)
			}
			rep = reply{Problems: rows}
		case "requirements":
			rep = reply{Requirements: nexusPageRequirements(req.Game, req.ModID)}
		case "":
			if err := open(req.Link); err != nil {
				rep = reply{Error: err.Error()}
			}
		default:
			rep = reply{Error: fmt.Sprintf("unknown request type %q", req.Type)}
		}
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
	if err != nil || !mortarRunning(dataDir) {
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
	if err := runningControlCall(dataDir, "modProblems", map[string]any{
		"game": info.ID, "profile": profileID, "modId": modID,
	}, &rows, time.Second); err != nil {
		return []modProblem{}
	}
	return rows
}

func runningControlCall(dir, method string, params map[string]any, out any, timeout time.Duration) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	b, err := root.ReadFile("control.json")
	_ = root.Close()
	if err != nil {
		return err
	}
	var discovery struct {
		Port  int    `json:"port"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(b, &discovery); err != nil {
		return err
	}
	conn, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(
		context.Background(),
		"tcp",
		net.JoinHostPort("127.0.0.1", strconv.Itoa(discovery.Port)),
	)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	req, err := json.Marshal(map[string]any{"token": discovery.Token, "method": method, "params": params})
	if err != nil {
		return err
	}
	if _, err := conn.Write(append(req, '\n')); err != nil {
		return err
	}
	var rep struct {
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(conn, 16<<20)).Decode(&rep); err != nil {
		return err
	}
	if rep.Error != "" {
		return errors.New(rep.Error)
	}
	if out == nil || len(rep.Result) == 0 {
		return nil
	}
	return json.Unmarshal(rep.Result, out)
}

// brokenNexusModIDs lists the Nexus pages SMAPI's compatibility list marks broken for the game version last played,
// from the copy Mortar cached for its Problems check.
func brokenNexusModIDs(domain string) []int {
	info, ok := components.BundledGameByNexusDomain(domain)
	if !ok || info.ID != "stardew" {
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
	if err != nil || !mortarRunning(dataDir) {
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
			add(info.Nexus.LoaderModID)
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
	if err != nil || !mortarRunning(dataDir) {
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
				name = entry.Mods[0].UniqueID
			}
			if entry.Mods[0].Version != "" {
				v := entry.Mods[0].Version
				version = &v
				installed = v
			}
		}
		pageVer, files := cachedNexusDetails(root, entry.Source.ModID)
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

func activeNexusConnected(domain string) bool {
	info, ok := components.BundledGameByNexusDomain(domain)
	if !ok {
		return false
	}
	dataDir, err := datadir.Dir()
	if err != nil || !mortarRunning(dataDir) {
		return false
	}
	store, err := settings.Open()
	if err != nil {
		return false
	}
	cur := store.Get()
	if conn, err := cur.Lookup("extensionConnection"); err == nil && conn == settings.ExtensionOff {
		return false
	}
	profileID := cur.LastProfile[info.ID]
	return profileID != "" && filepath.Base(profileID) == profileID
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
	if err != nil || !mortarRunning(dataDir) {
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
	pageVer, files := cachedNexusDetails(root, modID)
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
		var targets []string
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
				if m.UniqueID != "" {
					targets = append(targets, m.UniqueID)
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

func cachedNexusDetails(root *os.Root, modID int) (string, []cachedNexusFile) {
	c := &meta.Client{CacheDir: filepath.Join(root.Name(), "cache")}
	d, ok := meta.Peek[nexussvc.Details](c, nexussvc.DetailsName(modID))
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

func mortarRunning(dataDir string) bool {
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		return false
	}
	data, err := root.ReadFile("control.json")
	_ = root.Close()
	if err != nil {
		return false
	}
	var discovery struct {
		Port int `json:"port"`
	}
	if json.Unmarshal(data, &discovery) != nil || discovery.Port < 1 || discovery.Port > 65535 {
		return false
	}
	conn, err := (&net.Dialer{Timeout: time.Second}).DialContext(
		context.Background(),
		"tcp",
		net.JoinHostPort("127.0.0.1", strconv.Itoa(discovery.Port)),
	)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
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
		m["allowed_origins"] = []string{ChromeOrigin}
	}
	return json.MarshalIndent(m, "", "  ")
}

// accentColor reads only the accent from Mortar's settings file; the native host never writes Mortar's data, so it
// does not go through settings.Open, which moves a corrupt file aside.
func accentColor() string {
	var s struct {
		Accent string `json:"accent"`
	}
	if dir, err := datadir.Dir(); err == nil {
		if b, err := fsx.ReadFile(filepath.Join(dir, "settings.json")); err == nil {
			_ = json.Unmarshal(b, &s)
		}
	}
	return settings.AccentColor(s.Accent)
}
