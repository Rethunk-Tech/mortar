package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	defaultWindowWidth  = 1280
	defaultWindowHeight = 720
	minWindowWidth      = 768
	minWindowHeight     = 432
	minVisible          = 80
	// The play prompt fits one launch dialog (480px wide) with its margins.
	promptWindowWidth  = 560
	promptWindowHeight = 480
)

type windowGeom struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
	// Max reopens the window maximised; X, Y, W and H stay the size it returns to when restored.
	Max bool `json:"max,omitempty"`
}

type screenRect struct {
	X, Y, Width, Height int
}

func windowGeomPath(dataDir string) string {
	return filepath.Join(dataDir, "window.json")
}

func loadWindowGeom(dataDir string, screens []screenRect) (windowGeom, bool) {
	b, err := fsx.ReadFile(windowGeomPath(dataDir))
	if err != nil {
		return windowGeom{}, false
	}
	var g windowGeom
	if json.Unmarshal(b, &g) != nil || g.W <= 0 || g.H <= 0 {
		return windowGeom{}, false
	}
	g.X, g.Y, g.W, g.H = clampWindow(g.X, g.Y, g.W, g.H, screens)
	return g, true
}

// closingGeom is what to remember when the window closes: its bounds when it is a normal window, or, when it is
// maximised, the last normal bounds kept from before with Max set, so the next start neither opens a window the size
// of the screen nor forgets that it was maximised.
func closingGeom(prev windowGeom, havePrev, maximised bool, cur windowGeom) windowGeom {
	if !maximised {
		cur.Max = false
		return cur
	}
	if !havePrev {
		prev = windowGeom{W: defaultWindowWidth, H: defaultWindowHeight}
	}
	prev.Max = true
	return prev
}

func saveWindowGeom(dataDir string, g windowGeom) {
	_ = os.MkdirAll(dataDir, 0o700)
	_ = datadir.WriteJSON(windowGeomPath(dataDir), g)
}

func clampWindow(x, y, w, h int, screens []screenRect) (int, int, int, int) {
	if w < minWindowWidth {
		w = minWindowWidth
	}
	if h < minWindowHeight {
		h = minWindowHeight
	}
	if len(screens) == 0 {
		return x, y, w, h
	}
	for _, s := range screens {
		if windowVisible(x, y, w, h, s) {
			return x, y, w, h
		}
	}
	s := screens[0]
	return s.X, s.Y, w, h
}

func windowVisible(x, y, w, h int, s screenRect) bool {
	ox := max(x, s.X)
	oy := max(y, s.Y)
	ox2 := min(x+w, s.X+s.Width)
	oy2 := min(y+h, s.Y+s.Height)
	return ox2-ox >= minVisible && oy2-oy >= minVisible
}

func screensOf(app *application.App) []screenRect {
	if app == nil || app.Screen == nil {
		return nil
	}
	var out []screenRect
	for _, s := range app.Screen.GetAll() {
		if s == nil {
			continue
		}
		r := s.WorkArea
		if r.Width == 0 || r.Height == 0 {
			r = s.Bounds
		}
		out = append(out, screenRect{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height})
	}
	return out
}
