package nativehost

import (
	"cmp"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
)

type requirementItem struct {
	Name     string `json:"name"`
	ModID    int    `json:"modId,omitempty"`
	Present  bool   `json:"present"`
	External bool   `json:"external,omitempty"`
}

func nexusPageRequirements(domain string, modID int) []requirementItem {
	if modID < 1 {
		return nil
	}
	info, ok := components.GameByNexusDomain(domain)
	if !ok {
		return nil
	}
	dataDir, err := datadir.Dir()
	if err != nil {
		return nil
	}
	// The requirements ride on the page data Mortar keeps for the mod: its full details, else the batched page.
	cache := &meta.Client{}
	d, ok := nexussvc.PeekDetails(cache, domain, modID)
	if !ok {
		return nil
	}
	items := requirementItems(d.Page.Requirements)
	if len(items) == 0 {
		return nil
	}
	return markRequirementPresence(items, activeProfileNexusIDs(dataDir, info))
}

func requirementItems(reqs []nexusmods.Requirement) []requirementItem {
	items := make([]requirementItem, 0, len(reqs))
	for _, req := range reqs {
		name := strings.TrimSpace(req.Name)
		if req.External || req.ModID < 1 {
			name = cmp.Or(name, strings.TrimSpace(req.Notes))
			if name == "" {
				continue
			}
			items = append(items, requirementItem{Name: name, External: true})
			continue
		}
		name = cmp.Or(name, strconv.Itoa(req.ModID))
		items = append(items, requirementItem{Name: name, ModID: req.ModID})
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

func markRequirementPresence(items []requirementItem, installed []int) []requirementItem {
	have := make(map[int]struct{}, len(installed))
	for _, id := range installed {
		if id > 0 {
			have[id] = struct{}{}
		}
	}
	out := make([]requirementItem, len(items))
	copy(out, items)
	for i := range out {
		if out[i].External || out[i].ModID < 1 {
			continue
		}
		_, out[i].Present = have[out[i].ModID]
	}
	return out
}

func activeProfileNexusIDs(dataDir string, info components.GameInfo) []int {
	profile, ok := readOpenProfile(dataDir, info)
	if !ok {
		return nil
	}
	ids := []int{}
	seen := map[int]struct{}{}
	for _, entry := range profile.Entries {
		if entry.Source.Kind == "nexus" && entry.Source.ModID > 0 {
			if _, exists := seen[entry.Source.ModID]; !exists {
				seen[entry.Source.ModID] = struct{}{}
				ids = append(ids, entry.Source.ModID)
			}
		}
	}
	return ids
}
