package profile

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// MaxDescription caps a profile's short description, in characters.
const MaxDescription = 280

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

func cleanDescription(description string) (string, error) {
	description = strings.TrimSpace(description)
	if n := utf8.RuneCountInString(description); n > MaxDescription {
		return "", fmt.Errorf("description is longer than %d characters", MaxDescription)
	}
	return description, nil
}

func cleanAppearance(color, icon, description string) (string, string, string, error) {
	color = strings.TrimSpace(color)
	icon = strings.TrimSpace(icon)
	if !validColor(color) {
		return "", "", "", fmt.Errorf("unknown profile color %q", color)
	}
	if !validIcon(icon) {
		return "", "", "", fmt.Errorf("unknown profile icon %q", icon)
	}
	description, err := cleanDescription(description)
	if err != nil {
		return "", "", "", err
	}
	return color, icon, description, nil
}

func sanitizeAppearance(p *Profile) {
	if !validColor(p.Color) {
		p.Color = ""
	}
	if !validIcon(p.Icon) {
		p.Icon = ""
	}
	if d, err := cleanDescription(p.Description); err != nil {
		p.Description = ""
	} else {
		p.Description = d
	}
}

// SetAppearance replaces a profile's colour, icon and description. It never touches mods/, so a running
// game does not block it.
func (s *Store) SetAppearance(game, id, color, icon, description string) (Profile, error) {
	color, icon, description, err := cleanAppearance(color, icon, description)
	if err != nil {
		return Profile{}, err
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		p.Color = color
		p.Icon = icon
		p.Description = description
		return nil
	})
}
