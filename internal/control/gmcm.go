package control

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/gmcm"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func (s *Services) modsMenu(p Params) (any, error) {
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return nil, err
	}
	if len(p.IDs) == 0 {
		return nil, fmt.Errorf("mods menu needs a mod")
	}
	uid := typedID(p.IDs[0])
	if p.Value != "" {
		return s.setGmcm(p.Game, prof.ID, uid, p.Value)
	}
	menu, err := s.Profiles.GmcmMenu(p.Game, prof.ID, uid)
	if err != nil {
		return nil, err
	}
	return formatMenu(menu), nil
}

func (s *Services) setGmcm(game, profile string, uid mod.ID, spec string) (any, error) {
	page, index, raw, err := gmcm.ParseSetFlag(spec)
	if err != nil {
		return nil, err
	}
	menu, err := s.Profiles.GmcmMenu(game, profile, uid)
	if err != nil {
		return nil, err
	}
	edit, err := gmcm.EditFromCapture(menu, page, index, raw)
	if err != nil {
		return nil, err
	}
	pending, err := s.Profiles.PendingGmcm(game, profile, uid)
	if err != nil {
		return nil, err
	}
	edits := replaceEdit(pending.Edits, edit)
	if err := s.Profiles.SetGmcmEdits(game, profile, uid, edits); err != nil {
		return nil, err
	}
	return edits, nil
}

func replaceEdit(cur []gmcm.Edit, next gmcm.Edit) []gmcm.Edit {
	out := make([]gmcm.Edit, 0, len(cur)+1)
	replaced := false
	for _, e := range cur {
		if e.Page == next.Page && e.Index == next.Index {
			out = append(out, next)
			replaced = true
			continue
		}
		out = append(out, e)
	}
	if !replaced {
		out = append(out, next)
	}
	return out
}

func formatMenu(menu gmcm.Capture) []string {
	var lines []string
	for _, page := range menu.Pages {
		for _, opt := range page.Options {
			lines = append(lines, fmt.Sprintf("%s/%d %s %v", page.ID, opt.Index, opt.Name, opt.Value))
		}
	}
	return lines
}
