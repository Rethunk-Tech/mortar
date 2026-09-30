// Package modmenu is the menu of a mod: one action list feeds the card's ⋯ menu (through Service) and the native
// right-click menu (through Register), so they never drift.
package modmenu

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Events the native menu emits. Details and Remove need the window, so the frontend acts on them; Changed carries
// the profile after a switch, and Failed the error text of an action that could not run.
const (
	DetailsEvent = "mod:details"
	RemoveEvent  = "mod:remove"
	ChangedEvent = "mods:changed"
	FailedEvent  = "mod:failed"
)

// Target names one mod: the profile it is in, its entry and its UniqueID. It is the context data of the menu.
type Target struct {
	Game     string `json:"game"`
	Profile  string `json:"profile"`
	Key      string `json:"key"`
	UniqueID string `json:"uniqueId"`
}

// ParseTarget reads the context data the frontend attached to the element.
func ParseTarget(data string) (Target, error) {
	var t Target
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return Target{}, fmt.Errorf("mod menu data: %w", err)
	}
	if t.Game == "" || t.Profile == "" || t.Key == "" || t.UniqueID == "" {
		return Target{}, errors.New("mod menu data: game, profile, key and uniqueId are required")
	}
	return t, nil
}

// State is what decides which actions a mod's menu offers.
type State struct {
	Enabled   bool
	HasPage   bool
	Removable bool
}

// The actions, in menu order. Remove is set apart by a separator.
const (
	Toggle  = "toggle"
	Details = "details"
	Nexus   = "nexus"
	Files   = "files"
	Remove  = "remove"
)

// Actions lists the menu's actions for a mod in state s.
func Actions(s State) []string {
	out := []string{Toggle, Details}
	if s.HasPage {
		out = append(out, Nexus)
	}
	out = append(out, Files)
	if s.Removable {
		out = append(out, Remove)
	}
	return out
}

func flag(on bool, yes, no string) string {
	if on {
		return yes
	}
	return no
}

// MenuID names the registered native menu for state s. The frontend builds the same name.
func MenuID(s State) string {
	return "mod-menu-" + flag(s.Enabled, "on", "off") + "-" + flag(s.HasPage, "page", "nopage") + "-" +
		flag(s.Removable, "remove", "keep")
}

func states() []State {
	var out []State
	for _, enabled := range []bool{true, false} {
		for _, page := range []bool{true, false} {
			for _, removable := range []bool{true, false} {
				out = append(out, State{Enabled: enabled, HasPage: page, Removable: removable})
			}
		}
	}
	return out
}

// Service exposes the action list to the frontend.
type Service struct{}

// Actions is the list of actions for a mod in the given state.
func (Service) Actions(enabled, hasPage, removable bool) []string {
	return Actions(State{Enabled: enabled, HasPage: hasPage, Removable: removable})
}

// Backend is what the native menu's own actions call.
type Backend struct {
	SetEnabled func(t Target, enabled bool) (profile.Profile, error)
	ShowFiles  func(t Target) error
	PageURL    func(t Target) (string, error)
	OpenURL    func(url string) error
	Emit       func(name string, data ...any) bool
}

func label(action string, s State) string {
	switch action {
	case Toggle:
		return flag(s.Enabled, "Disable", "Enable")
	case Details:
		return "More details"
	case Nexus:
		return "Open page"
	case Files:
		return "Show files"
	default:
		return "Remove"
	}
}

func (b Backend) run(ctx *application.Context, s State, action string) {
	t, err := ParseTarget(ctx.ContextMenuData())
	if err == nil {
		err = b.act(t, s, action)
	}
	if err != nil {
		b.Emit(FailedEvent, err.Error())
	}
}

func (b Backend) act(t Target, s State, action string) error {
	switch action {
	case Toggle:
		p, err := b.SetEnabled(t, !s.Enabled)
		if err == nil {
			b.Emit(ChangedEvent, p)
		}
		return err
	case Details:
		b.Emit(DetailsEvent, t)
	case Nexus:
		url, err := b.PageURL(t)
		if err != nil {
			return err
		}
		return b.OpenURL(url)
	case Files:
		return b.ShowFiles(t)
	case Remove:
		b.Emit(RemoveEvent, t)
	}
	return nil
}

// Register adds every state's native menu to the app.
func Register(app *application.App, b Backend) {
	for _, s := range states() {
		menu := app.ContextMenu.New()
		for _, action := range Actions(s) {
			if action == Remove {
				menu.AddSeparator()
			}
			menu.Add(label(action, s)).OnClick(func(ctx *application.Context) { b.run(ctx, s, action) })
		}
		app.ContextMenu.Add(MenuID(s), menu)
	}
}
