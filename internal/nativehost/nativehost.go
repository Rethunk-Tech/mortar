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
	Type string `json:"type"`
	Link string `json:"link"`
	Game string `json:"game"`
}

type reply struct {
	OK     bool   `json:"ok,omitempty"`
	Error  string `json:"error,omitempty"`
	ModIDs *[]int `json:"modIds,omitempty"`
}

// Serve answers messages from r until it closes, handing each message's link to open.
func Serve(r io.Reader, w io.Writer, open func(link string) error) error {
	return serve(r, w, open, activeNexusModIDs)
}

func serve(r io.Reader, w io.Writer, open func(link string) error, installed func(game string) []int) error {
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
			rep = reply{ModIDs: &ids}
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

func activeNexusModIDs(domain string) []int {
	ids := []int{}
	if !strings.EqualFold(domain, "stardewvalley") {
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
	current := store.Get()
	if current.LastGame != "stardew" {
		return ids
	}
	profileID := current.LastProfile["stardew"]
	if profileID == "" || filepath.Base(profileID) != profileID {
		return ids
	}
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		return ids
	}
	data, err := root.ReadFile(filepath.Join("profiles", "stardew", profileID, "profile.json"))
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
