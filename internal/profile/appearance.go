package profile

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// ProfileColors are the palette tokens a profile may store.
var ProfileColors = []string{
	"rose",
	"orange",
	"gold",
	"lime",
	"teal",
	"sky",
	"violet",
	"pink",
}

// ProfileIcons are the Lucide names a profile may store.
var ProfileIcons = []string{
	"sprout",
	"leaf",
	"wheat",
	"fish",
	"hammer",
	"pickaxe",
	"star",
	"heart",
	"mountain",
	"sun",
	"moon",
	"sparkles",
}

func validColor(color string) bool {
	return color == "" || slices.Contains(ProfileColors, color)
}

func validIcon(icon string) bool {
	return icon == "" || slices.Contains(ProfileIcons, icon)
}

func cleanAppearance(color, icon string) (string, string, error) {
	color = strings.TrimSpace(color)
	icon = strings.TrimSpace(icon)
	if !validColor(color) {
		return "", "", fmt.Errorf("unknown profile color %q", color)
	}
	if !validIcon(icon) {
		return "", "", fmt.Errorf("unknown profile icon %q", icon)
	}
	return color, icon, nil
}

func sanitizeAppearance(p *Profile) {
	if !validColor(p.Color) {
		p.Color = ""
	}
	if !validIcon(p.Icon) {
		p.Icon = ""
	}
}

// foldDescription moves a "description" key, which Mortar no longer writes, into the notes so nothing a player
// wrote is lost; the next write of the profile drops the key.
func foldDescription(raw []byte, p *Profile) {
	var old struct {
		Description string `json:"description"`
	}
	if json.Unmarshal(raw, &old) == nil {
		p.Notes = FoldDescription(old.Description, p.Notes)
	}
}

// FoldDescription puts a profile's old short description on top of its notes, a blank line between, unless the
// notes already hold it.
func FoldDescription(description, notes string) string {
	d := strings.TrimSpace(description)
	switch {
	case d == "" || strings.Contains(notes, d):
		return notes
	case strings.TrimSpace(notes) == "":
		return d
	default:
		return d + "\n\n" + notes
	}
}

// SetAppearance replaces a profile's colour and icon. It never touches mods/, so a running game does not block it.
func (s *Store) SetAppearance(game, id, color, icon string) (Profile, error) {
	color, icon, err := cleanAppearance(color, icon)
	if err != nil {
		return Profile{}, err
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		p.Color = color
		p.Icon = icon
		return nil
	})
}
