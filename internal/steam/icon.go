package steam

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"regexp"

	"github.com/Rethunk-Tech/go-safefs"
)

// iconName is the content hash Steam names an app's icon by, in the library cache and in steam/games.
var iconName = regexp.MustCompile(`^[0-9a-f]{40}\.jpg$`)

var pngMagic = []byte("\x89PNG\r\n\x1a\n")

// ClientIcon is appID's square icon: the largest image of the client icon Steam keeps in steam/games, else the small
// copy in its library cache; nil when Steam has neither.
func (s Steam) ClientIcon(appID string) []byte {
	if !filepath.IsLocal(appID) {
		return nil
	}
	cache := filepath.Join(s.Root, "appcache", "librarycache", appID)
	matches, _ := filepath.Glob(filepath.Join(cache, "*.jpg"))
	for _, jpg := range matches {
		name := filepath.Base(jpg)
		if !iconName.MatchString(name) {
			continue
		}
		if ico, err := safefs.ReadFile(filepath.Join(s.Root, "steam", "games", name[:40]+".ico")); err == nil {
			if png := largestPNG(ico); png != nil {
				return png
			}
		}
		if b, err := safefs.ReadFile(jpg); err == nil {
			return b
		}
	}
	return nil
}

// largestPNG is the widest PNG image in an .ico file, nil when it holds none (older icons store bitmaps only).
func largestPNG(ico []byte) []byte {
	if len(ico) < 6 || binary.LittleEndian.Uint16(ico[2:]) != 1 {
		return nil
	}
	count := int(binary.LittleEndian.Uint16(ico[4:]))
	var best []byte
	bestWidth := 0
	for i := range count {
		at := 6 + 16*i
		if at+16 > len(ico) {
			return best
		}
		width := int(ico[at])
		if width == 0 {
			width = 256
		}
		size := int(binary.LittleEndian.Uint32(ico[at+8:]))
		offset := int(binary.LittleEndian.Uint32(ico[at+12:]))
		if offset < 0 || size <= 0 || offset+size > len(ico) || !bytes.HasPrefix(ico[offset:offset+size], pngMagic) {
			continue
		}
		if width > bestWidth {
			best, bestWidth = ico[offset:offset+size], width
		}
	}
	return best
}
