package control

import (
	"errors"
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/modreport"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func (s *Services) modReport(gameID, profileID string, prof profile.Profile, p Params) (modreport.Result, error) {
	if len(p.IDs) == 0 {
		return modreport.Result{}, errors.New("mods report needs a mod")
	}
	refs, err := refsFor(prof, p.IDs[:1])
	if err != nil {
		return modreport.Result{}, err
	}
	ref := refs[0]
	_, inst, ok := prof.FindMod(ref.Key, ref.ID)
	if !ok {
		return modreport.Result{}, fmt.Errorf("profile has no mod %q", p.IDs[0])
	}
	logText, err := s.runLog(gameID, profileID, p.Run)
	if err != nil {
		return modreport.Result{}, err
	}
	domain := ""
	if info, ok := components.Game(gameID); ok {
		domain = info.NexusDomain()
	}
	src := entrySource(prof, ref.Key)
	in := modreport.Input{
		Game:          gameID,
		ModName:       inst.Name,
		ModVersion:    inst.Version,
		MortarVersion: s.Version,
		LogShareURL:   p.Value,
		Source:        src,
		NexusDomain:   domain,
	}
	if src.Kind == profile.KindGitHub {
		in.GitHubRepo = src.Repo
	}
	if src.Kind == profile.KindNexus {
		in.NexusModID = src.ModID
	}
	return modreport.BuildFromLog(logText.Text, inst.Name, in), nil
}

func entrySource(p profile.Profile, key string) profile.Source {
	for _, e := range p.Entries {
		if e.Key == key {
			return e.Source
		}
	}
	return profile.Source{}
}
