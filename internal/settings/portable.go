package settings

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const exportVersion = 1

// Portable is settings.json minus secrets and machine-specific or personal fields.
type Portable struct {
	Version                int      `json:"version"`
	Accent                 string   `json:"accent"`
	Background             string   `json:"background"`
	LastGame               string   `json:"lastGame"`
	BackupsKept            int      `json:"backupsKept"`
	ListColumns            []string `json:"listColumns"`
	ListSortColumn         string   `json:"listSortColumn"`
	ListSortDir            string   `json:"listSortDir"`
	ListGroupBy            string   `json:"listGroupBy"`
	CheckModUpdatesOnStart *bool    `json:"checkModUpdatesOnStart"`
	TellWhenSmapiOut       *bool    `json:"tellWhenSmapiOut"`
	KeepInTray             bool     `json:"keepInTray"`
	IncludeBetaReleases    bool     `json:"includeBetaReleases"`
	TipsSeen               []string `json:"tipsSeen"`
}

// Change is one field that import would replace.
type Change struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// ImportPreview is a validated export ready to apply, or empty Raw when the dialog was cancelled.
type ImportPreview struct {
	Raw     string   `json:"raw"`
	Changes []Change `json:"changes"`
}

var portableFields = []string{
	"accent", "background", "lastGame", "backupsKept",
	"listColumns", "listSortColumn", "listSortDir", "listGroupBy",
	"checkModUpdatesOnStart", "tellWhenSmapiOut", "keepInTray", "includeBetaReleases", "tipsSeen",
}

// MarshalExport writes a versioned JSON of s without secrets or machine-specific fields.
func MarshalExport(s Settings) ([]byte, error) {
	p := Portable{
		Version:                exportVersion,
		Accent:                 s.Accent,
		Background:             s.Background,
		LastGame:               s.LastGame,
		BackupsKept:            s.BackupsKept,
		ListColumns:            slices.Clone(s.ListColumns),
		ListSortColumn:         s.ListSortColumn,
		ListSortDir:            s.ListSortDir,
		ListGroupBy:            s.ListGroupBy,
		CheckModUpdatesOnStart: s.CheckModUpdatesOnStart,
		TellWhenSmapiOut:       s.TellWhenSmapiOut,
		KeepInTray:             s.KeepInTray,
		IncludeBetaReleases:    s.IncludeBetaReleases,
		TipsSeen:               slices.Clone(s.TipsSeen),
	}
	return json.MarshalIndent(p, "", "  ")
}

// ParseExport validates an export. Unknown fields are ignored; each portable value is sanitised as on load.
func ParseExport(b []byte) (Portable, map[string]struct{}, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return Portable{}, nil, err
	}
	if _, ok := raw["version"]; !ok {
		return Portable{}, nil, fmt.Errorf("missing version")
	}
	var p Portable
	if err := json.Unmarshal(b, &p); err != nil {
		return Portable{}, nil, err
	}
	if p.Version != exportVersion {
		return Portable{}, nil, fmt.Errorf("unsupported settings version %d", p.Version)
	}
	present := make(map[string]struct{}, len(raw))
	for k := range raw {
		present[k] = struct{}{}
	}
	overlay := Defaults()
	if _, ok := present["accent"]; ok {
		overlay.Accent = p.Accent
	}
	if _, ok := present["background"]; ok {
		overlay.Background = p.Background
	}
	if _, ok := present["lastGame"]; ok {
		overlay.LastGame = p.LastGame
	}
	if _, ok := present["backupsKept"]; ok {
		overlay.BackupsKept = p.BackupsKept
	}
	if _, ok := present["listColumns"]; ok {
		overlay.ListColumns = p.ListColumns
	}
	if _, ok := present["listSortColumn"]; ok {
		overlay.ListSortColumn = p.ListSortColumn
	}
	if _, ok := present["listSortDir"]; ok {
		overlay.ListSortDir = p.ListSortDir
	}
	if _, ok := present["listGroupBy"]; ok {
		overlay.ListGroupBy = p.ListGroupBy
	}
	if _, ok := present["checkModUpdatesOnStart"]; ok {
		overlay.CheckModUpdatesOnStart = p.CheckModUpdatesOnStart
	}
	if _, ok := present["tellWhenSmapiOut"]; ok {
		overlay.TellWhenSmapiOut = p.TellWhenSmapiOut
	}
	if _, ok := present["keepInTray"]; ok {
		overlay.KeepInTray = p.KeepInTray
	}
	if _, ok := present["includeBetaReleases"]; ok {
		overlay.IncludeBetaReleases = p.IncludeBetaReleases
	}
	if _, ok := present["tipsSeen"]; ok {
		overlay.TipsSeen = p.TipsSeen
	}
	overlay = sanitizePortable(overlay)
	if _, ok := present["accent"]; ok {
		p.Accent = overlay.Accent
	}
	if _, ok := present["background"]; ok {
		p.Background = overlay.Background
	}
	if _, ok := present["lastGame"]; ok {
		p.LastGame = overlay.LastGame
	}
	if _, ok := present["backupsKept"]; ok {
		p.BackupsKept = overlay.BackupsKept
	}
	if _, ok := present["listColumns"]; ok {
		p.ListColumns = overlay.ListColumns
	}
	if _, ok := present["listSortColumn"]; ok {
		p.ListSortColumn = overlay.ListSortColumn
	}
	if _, ok := present["listSortDir"]; ok {
		p.ListSortDir = overlay.ListSortDir
	}
	if _, ok := present["listGroupBy"]; ok {
		p.ListGroupBy = overlay.ListGroupBy
	}
	if _, ok := present["checkModUpdatesOnStart"]; ok {
		p.CheckModUpdatesOnStart = overlay.CheckModUpdatesOnStart
	}
	if _, ok := present["tellWhenSmapiOut"]; ok {
		p.TellWhenSmapiOut = overlay.TellWhenSmapiOut
	}
	if _, ok := present["keepInTray"]; ok {
		p.KeepInTray = overlay.KeepInTray
	}
	if _, ok := present["includeBetaReleases"]; ok {
		p.IncludeBetaReleases = overlay.IncludeBetaReleases
	}
	if _, ok := present["tipsSeen"]; ok {
		p.TipsSeen = overlay.TipsSeen
	}
	return p, present, nil
}

// ApplyExport copies sanitised portable fields that were present in the file onto cur.
func ApplyExport(cur *Settings, p Portable, present map[string]struct{}) {
	if _, ok := present["accent"]; ok {
		cur.Accent = p.Accent
	}
	if _, ok := present["background"]; ok {
		cur.Background = p.Background
	}
	if _, ok := present["lastGame"]; ok {
		cur.LastGame = p.LastGame
	}
	if _, ok := present["backupsKept"]; ok {
		cur.BackupsKept = p.BackupsKept
	}
	if _, ok := present["listColumns"]; ok {
		cur.ListColumns = slices.Clone(p.ListColumns)
	}
	if _, ok := present["listSortColumn"]; ok {
		cur.ListSortColumn = p.ListSortColumn
	}
	if _, ok := present["listSortDir"]; ok {
		cur.ListSortDir = p.ListSortDir
	}
	if _, ok := present["listGroupBy"]; ok {
		cur.ListGroupBy = p.ListGroupBy
	}
	if _, ok := present["checkModUpdatesOnStart"]; ok {
		cur.CheckModUpdatesOnStart = p.CheckModUpdatesOnStart
	}
	if _, ok := present["tellWhenSmapiOut"]; ok {
		cur.TellWhenSmapiOut = p.TellWhenSmapiOut
	}
	if _, ok := present["keepInTray"]; ok {
		cur.KeepInTray = p.KeepInTray
	}
	if _, ok := present["includeBetaReleases"]; ok {
		cur.IncludeBetaReleases = p.IncludeBetaReleases
	}
	if _, ok := present["tipsSeen"]; ok {
		cur.TipsSeen = slices.Clone(p.TipsSeen)
	}
}

func sanitizePortable(s Settings) Settings {
	if !slices.Contains(accents, s.Accent) {
		s.Accent = Defaults().Accent
	}
	if !slices.Contains(backgrounds, s.Background) {
		s.Background = Defaults().Background
	}
	if s.BackupsKept < MinBackupsKept || s.BackupsKept > MaxBackupsKept {
		s.BackupsKept = Defaults().BackupsKept
	}
	normalizeToggles(&s)
	normalizeList(&s)
	normalizeTips(&s)
	return s
}

func previewChanges(cur Settings, p Portable, present map[string]struct{}) []Change {
	next := cur
	ApplyExport(&next, p, present)
	var out []Change
	for _, field := range portableFields {
		if _, ok := present[field]; !ok {
			continue
		}
		from, to := fieldText(cur, field), fieldText(next, field)
		if from == to {
			continue
		}
		out = append(out, Change{Field: field, From: from, To: to})
	}
	if out == nil {
		out = []Change{}
	}
	return out
}

func fieldText(s Settings, field string) string {
	switch field {
	case "accent":
		return s.Accent
	case "background":
		return s.Background
	case "lastGame":
		return s.LastGame
	case "backupsKept":
		return strconv.Itoa(s.BackupsKept)
	case "listColumns":
		return strings.Join(s.ListColumns, ", ")
	case "listSortColumn":
		return s.ListSortColumn
	case "listSortDir":
		return s.ListSortDir
	case "listGroupBy":
		return s.ListGroupBy
	case "checkModUpdatesOnStart":
		return boolText(s.CheckModUpdatesOnStart)
	case "tellWhenSmapiOut":
		return boolText(s.TellWhenSmapiOut)
	case "keepInTray":
		return strconv.FormatBool(s.KeepInTray)
	case "includeBetaReleases":
		return strconv.FormatBool(s.IncludeBetaReleases)
	case "tipsSeen":
		return strings.Join(s.TipsSeen, ", ")
	default:
		return ""
	}
}

func boolText(v *bool) string {
	if v == nil {
		return ""
	}
	return strconv.FormatBool(*v)
}
