package nativehost

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/settings"
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
	info, ok := components.BundledGameByNexusDomain(domain)
	if !ok {
		return nil
	}
	dataDir, err := datadir.Dir()
	if err != nil {
		return nil
	}
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		return nil
	}
	defer func() { _ = root.Close() }()
	reqs, ok := meta.Peek[[]meta.Requirement](&meta.Client{}, meta.RequirementsCacheFile(modID))
	if !ok {
		return nil
	}
	items := requirementItems(reqs)
	if len(items) == 0 {
		return nil
	}
	return markRequirementPresence(items, activeProfileNexusIDs(root, info.ID))
}

func requirementItems(reqs []meta.Requirement) []requirementItem {
	items := make([]requirementItem, 0, len(reqs))
	for _, req := range reqs {
		name := strings.TrimSpace(req.Name)
		if req.ModID < 1 {
			if name == "" {
				name = strings.TrimSpace(req.Notes)
			}
			if name == "" {
				continue
			}
			items = append(items, requirementItem{Name: name, External: true})
			continue
		}
		if name == "" {
			name = strconv.Itoa(req.ModID)
		}
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

func activeProfileNexusIDs(root *os.Root, gameID string) []int {
	store, err := settings.Open()
	if err != nil {
		return nil
	}
	profileID := store.Get().LastProfile[gameID]
	if profileID == "" || filepath.Base(profileID) != profileID {
		return nil
	}
	data, err := root.ReadFile(filepath.Join("profiles", gameID, profileID, "profile.json"))
	if err != nil {
		return nil
	}
	var profile struct {
		Entries []struct {
			Source struct {
				Kind  string `json:"kind"`
				ModID int    `json:"modId"`
			} `json:"source"`
		} `json:"entries"`
	}
	if json.Unmarshal(data, &profile) != nil {
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
