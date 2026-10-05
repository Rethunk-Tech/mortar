package configsvc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/gmcm"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// gmcmFileName names the in-game menu among a mod's config files; no config file is called that.
const gmcmFileName = "In-game menu"

// gmcmKey is an option's key within the menu: its page and its place on the page, which is how the bridge finds it.
func gmcmKey(page string, index int) string { return page + "/" + strconv.Itoa(index) }

func parseGmcmKey(key string) (string, int, error) {
	page, rest, ok := strings.CutLast(key, "/")
	if !ok {
		return "", 0, fmt.Errorf("%q is not an in-game menu option", key)
	}
	index, err := strconv.Atoi(rest)
	return page, index, err
}

// gmcmSchema lays the captured menu out as sections: each page, split again at the menu's own section titles. An
// option shows its pending value when one waits for the next start, and the captured value is its default, so Reset
// drops the pending edit.
func (s *Service) gmcmSchema(game, profileID string, uniqueID mod.ID, f ConfigFile) (Schema, error) {
	dir, err := s.Profiles.ProfileDir(game, profileID)
	if err != nil {
		return Schema{}, err
	}
	menu, err := gmcm.ReadCapture(dir, uniqueID)
	if err != nil {
		return Schema{}, err
	}
	pending, err := gmcm.ReadPending(dir, uniqueID)
	if err != nil {
		return Schema{}, err
	}
	result, _ := gmcm.ReadResult(dir, uniqueID)
	return gmcmSections(menu, pending, result, f), nil
}

func gmcmSections(menu gmcm.Capture, pending gmcm.Pending, result gmcm.Result, f ConfigFile) Schema {
	waiting := map[string]gmcm.Edit{}
	for _, e := range pending.Edits {
		waiting[gmcmKey(e.Page, e.Index)] = e
	}
	failed := map[string]string{}
	for _, sk := range result.Skipped {
		failed[gmcmKey(sk.Edit.Page, sk.Edit.Index)] = sk.Reason
	}
	out := Schema{File: f, Sections: []Section{}}
	used := map[string]int{}
	start := func(name string) {
		used[name]++
		if n := used[name]; n > 1 {
			name = fmt.Sprintf("%s (%d)", name, n)
		}
		out.Sections = append(out.Sections, Section{Name: name})
	}
	for _, p := range menu.Pages {
		pageName := p.Title
		if pageName == "" && p.ID != "" {
			pageName = p.ID
		}
		start(pageName)
		for _, o := range p.Options {
			switch o.Kind {
			case "sectionTitle", "sectionSubHeader":
				name := o.Name
				if pageName != "" {
					name = pageName + " › " + o.Name
				}
				if last := &out.Sections[len(out.Sections)-1]; len(last.Entries) == 0 {
					used[last.Name]--
					out.Sections = out.Sections[:len(out.Sections)-1]
				}
				start(name)
				continue
			case "paragraph", "pageLink":
				continue
			}
			key := gmcmKey(p.ID, o.Index)
			e := gmcmEntry(o)
			e.Key = key
			if edit, ok := waiting[key]; ok {
				e.Value, e.Pending = gmcm.Text(edit.Value), true
			}
			if reason, ok := failed[key]; ok {
				e.Note = reason
			}
			last := &out.Sections[len(out.Sections)-1]
			last.Entries = append(last.Entries, e)
		}
	}
	out.Sections = dropEmpty(out.Sections)
	return out
}

func dropEmpty(sections []Section) []Section {
	kept := sections[:0]
	for _, s := range sections {
		if len(s.Entries) > 0 {
			kept = append(kept, s)
		}
	}
	return kept
}

func gmcmEntry(o gmcm.Option) Entry {
	value := gmcm.Text(o.Value)
	e := Entry{Label: o.Name, Value: value, Default: value, HasDefault: true, Description: o.Tooltip, Min: o.Min, Max: o.Max}
	switch o.Kind {
	case "bool":
		e.Type = TypeBool
	case "int":
		e.Type = TypeInt
	case "float":
		e.Type = TypeFloat
	case "choice":
		e.Type = TypeEnum
		for _, c := range o.Choices {
			e.Values = append(e.Values, gmcm.Text(c.Value))
			e.Labels = append(e.Labels, c.Label)
		}
	default:
		e.Type = TypeString
	}
	// The bridge sets plain values only; a mod's own widgets (colours, images, custom drawing) stay the game's to change.
	if !o.Editable || o.Kind == "color" || o.Kind == "image" || o.Kind == "complex" {
		e.Type, e.ReadOnly, e.Value, e.Default = TypeString, true, "", ""
	}
	return e
}

func (s *Service) writeGmcm(game, profileID string, uniqueID mod.ID, edits []edit) error {
	if s.SetGmcm == nil {
		return errors.New("in-game menu edits are not wired")
	}
	for _, e := range edits {
		page, index, err := parseGmcmKey(e.key)
		if err != nil {
			return err
		}
		if err := s.SetGmcm(game, profileID, uniqueID, page, index, e.value); err != nil {
			return err
		}
	}
	return nil
}
