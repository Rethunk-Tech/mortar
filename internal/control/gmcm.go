package control

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/gmcm"
)

func (s *Services) modsMenu(p Params) (any, error) {
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return nil, err
	}
	if len(p.UniqueIDs) == 0 {
		return nil, fmt.Errorf("mods menu needs a mod")
	}
	uid := p.UniqueIDs[0]
	if p.Value != "" {
		return s.setGmcm(p.Game, prof.ID, uid, p.Value)
	}
	cap, err := s.Profiles.GmcmMenu(p.Game, prof.ID, uid)
	if err != nil {
		return nil, err
	}
	return formatMenu(cap), nil
}

func (s *Services) setGmcm(game, profile, uid, spec string) (any, error) {
	page, index, raw, err := gmcm.ParseSetFlag(spec)
	if err != nil {
		return nil, err
	}
	cap, err := s.Profiles.GmcmMenu(game, profile, uid)
	if err != nil {
		return nil, err
	}
	edit, err := gmcm.EditFromCapture(cap, page, index, raw)
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

func formatMenu(cap gmcm.Capture) []string {
	var lines []string
	for _, page := range cap.Pages {
		for _, opt := range page.Options {
			lines = append(lines, fmt.Sprintf("%s/%d %s %v", page.ID, opt.Index, opt.Name, opt.Value))
		}
	}
	return lines
}
