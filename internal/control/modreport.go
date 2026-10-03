package control

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/components"
	"github.com/Rethunk-AI/mortar/internal/modreport"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

// ModReport is plain text plus an author URL from mods.report.
type ModReport = modreport.Result

func (s *Services) modReport(gameID, profileID string, prof profile.Profile, p Params) (ModReport, error) {
	if len(p.UniqueIDs) == 0 {
		return ModReport{}, errors.New("mods report needs a mod")
	}
	refs, err := refsFor(prof, p.UniqueIDs[:1])
	if err != nil {
		return ModReport{}, err
	}
	ref := refs[0]
	inst, ok := installedMod(prof, ref.Key, ref.UniqueID)
	if !ok {
		return ModReport{}, fmt.Errorf("profile has no mod %q", p.UniqueIDs[0])
	}
	logText, err := s.runLog(gameID, profileID, p.Run)
	if err != nil {
		return ModReport{}, err
	}
	domain := ""
	if info, ok := components.BundledGame(gameID); ok {
		domain = info.Nexus.Domain
	}
	src := entrySource(prof, ref.Key)
	in := modreport.Input{
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

func installedMod(p profile.Profile, key, uniqueID string) (profile.EntryMod, bool) {
	for _, e := range p.Entries {
		if e.Key != key {
			continue
		}
		for _, m := range e.Mods {
			if strings.EqualFold(m.UniqueID, uniqueID) {
				return m, true
			}
		}
	}
	return profile.EntryMod{}, false
}

func entrySource(p profile.Profile, key string) profile.Source {
	for _, e := range p.Entries {
		if e.Key == key {
			return e.Source
		}
	}
	return profile.Source{}
}
