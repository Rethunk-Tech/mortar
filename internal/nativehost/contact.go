package nativehost

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// ContactFile is the data-folder file recording the last time a browser's extension talked to Mortar.
const ContactFile = "extension-contact.json"

// contactEvery bounds how often the host rewrites ContactFile; a burst of messages is one stamp.
const contactEvery = time.Minute

// Contact is the last browser extension contact. The zero value means no browser has connected.
type Contact struct {
	Browser  string    `json:"browser"`
	LastSeen time.Time `json:"lastSeen"`
	// Mismatch is ExtensionTooOld or ExtensionTooNew when Mortar refused the extension's protocol.
	Mismatch string `json:"mismatch,omitempty"`
}

var (
	contactMu    sync.Mutex
	contactWrote Contact
)

// browserName names the browser that started the host: Firefox passes its extension id, and a Chromium browser is
// named by its own process where the platform shows it.
func browserName(args []string) string {
	if len(args) > 1 && args[1] == FirefoxID {
		return "Firefox"
	}
	if exe, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(os.Getppid()), "exe")); err == nil {
		if name := strings.TrimSuffix(filepath.Base(exe), ".exe"); name != "" {
			return name
		}
	}
	return "Chromium"
}

func recordContact(browser, mismatch string, now time.Time) {
	contactMu.Lock()
	defer contactMu.Unlock()
	if now.Sub(contactWrote.LastSeen) < contactEvery && contactWrote.Mismatch == mismatch {
		return
	}
	dir, err := datadir.Dir()
	if err != nil {
		return
	}
	c := Contact{Browser: browser, LastSeen: now.UTC(), Mismatch: mismatch}
	if datadir.WriteJSON(filepath.Join(dir, ContactFile), c) == nil {
		contactWrote = c
	}
}

// LastContact reads the last recorded contact from the data folder; it is empty before any browser connected.
func LastContact() Contact {
	var c Contact
	if dir, err := datadir.Dir(); err == nil {
		_, _ = datadir.ReadJSON(filepath.Join(dir, ContactFile), &c)
	}
	return c
}
