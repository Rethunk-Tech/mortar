package control

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// ModExtraFile is one linked extra store item on a mod entry.
type ModExtraFile struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

func (s *Services) modExtraFiles(prof profile.Profile, modID string) ([]ModExtraFile, error) {
	e, _, ok := prof.FindMod("", modID)
	if !ok {
		return nil, fmt.Errorf("profile %s has no mod %q", prof.Name, modID)
	}
	var files []nexus.File
	if e.Source.Kind == profile.KindNexus && e.Source.ModID > 0 && s.Nexus != nil {
		if d, ok := s.Nexus.CachedDetails([]int{e.Source.ModID})[e.Source.ModID]; ok {
			files = d.Files
		}
	}
	out := make([]ModExtraFile, 0, len(e.ExtraStoreKeys))
	for _, key := range e.ExtraStoreKeys {
		out = append(out, ModExtraFile{Key: key, Label: profile.ExtraFileLabel(e, key, files)})
	}
	return out, nil
}
