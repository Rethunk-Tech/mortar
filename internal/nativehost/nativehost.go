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
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/components"
	"github.com/Rethunk-AI/mortar/internal/datadir"
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
	OK        bool           `json:"ok,omitempty"`
	Error     string         `json:"error,omitempty"`
	Connected bool           `json:"connected"`
	ModIDs    *[]int         `json:"modIds,omitempty"`
	Open      *modInProfile  `json:"open,omitempty"`
	Others    []modInProfile `json:"others,omitempty"`
	Problems  []modProblem   `json:"problems,omitempty"`
}

type modProblem struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type modInProfile struct {
	Profile string  `json:"profile"`
	Version *string `json:"version"`
	FileID  int     `json:"fileId,omitempty"`
}

// Serve answers messages from r until it closes, handing each message's link to open.
func Serve(r io.Reader, w io.Writer, open func(link string) error) error {
	return serveWithConnection(r, w, open, activeNexusModIDs, nexusModProfiles, activeNexusConnected, nexusModProblems)
}

func serve(r io.Reader, w io.Writer, open func(link string) error, installed func(game string) []int, mod func(game string, modID int) (modInProfile, []modInProfile), problem ...func(game string, modID int) []modProblem) error {
	return serveWithConnection(r, w, open, installed, mod, nil, problem...)
}

func serveWithConnection(r io.Reader, w io.Writer, open func(link string) error, installed func(game string) []int, mod func(game string, modID int) (modInProfile, []modInProfile), connected func(string) bool, problem ...func(game string, modID int) []modProblem) error {
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
		case "mod":
			openProfile, others := mod(req.Game, req.ModID)
			rep = reply{Open: &openProfile, Others: others}
		case "modProblems":
			var rows []modProblem
			if len(problem) > 0 && problem[0] != nil {
				rows = problem[0](req.Game, req.ModID)
			}
			rep = reply{Problems: rows}
		case "":
			if err := open(req.Link); err != nil {
				rep = reply{Error: err.Error()}
			}
		default:
			rep = reply{Error: fmt.Sprintf("unknown request type %q", req.Type)}
		}
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
				ModID int    `json:"modId"`
			} `json:"source"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return ids
	}
	seen := make(map[int]struct{}, len(profile.Entries))
	for _, entry := range profile.Entries {
		if entry.Source.Kind == "nexus" && entry.Source.ModID > 0 {
			if _, exists := seen[entry.Source.ModID]; !exists {
				seen[entry.Source.ModID] = struct{}{}
				ids = append(ids, entry.Source.ModID)
			}
		}
	}
	return ids
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
	profileID := store.Get().LastProfile[info.ID]
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
			Name    string `json:"name"`
			Hidden  bool   `json:"hidden"`
			Entries []struct {
				Source struct {
					Kind   string `json:"kind"`
					ModID  int    `json:"modId"`
					FileID int    `json:"fileId"`
				} `json:"source"`
				Mods []struct {
					Version string `json:"version"`
				} `json:"mods"`
			} `json:"entries"`
		}
		data, err := root.ReadFile(filepath.Join("profiles", info.ID, dir.Name(), "profile.json"))
		if err != nil || json.Unmarshal(data, &profile) != nil || profile.Hidden {
			continue
		}
		var version *string
		var fileID int
		for _, entry := range profile.Entries {
			if entry.Source.Kind != "nexus" || entry.Source.ModID != modID {
				continue
			}
			if len(entry.Mods) > 0 && entry.Mods[0].Version != "" {
				v := entry.Mods[0].Version
				version = &v
			}
			fileID = entry.Source.FileID
			break
		}
		found := modInProfile{Profile: profile.Name, Version: version, FileID: fileID}
		if dir.Name() == openID {
			openProfile = found
		} else {
			others = append(others, found)
		}
	}
	return openProfile, others
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
