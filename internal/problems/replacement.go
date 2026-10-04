package problems

import (
	"cmp"
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

var (
	nexusModURL = regexp.MustCompile(`(?i)https?://(?:www\.)?nexusmods\.com/stardewvalley/mods/(\d+)`)
	githubURL   = regexp.MustCompile(`(?i)https?://(?:www\.)?github\.com/([^/\s]+/[^/\s#?]+)`)
	uniqueID    = regexp.MustCompile(`\b([A-Za-z][A-Za-z0-9_]*\.[A-Za-z][A-Za-z0-9_]*)\b`)
)

// replacementFromSummary picks a mod to install when SMAPI's compatibility summary names one.
func replacementFromSummary(ctx context.Context, m Meta, dependentKeys []string, summary string) *Ref {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil
	}
	if parts := nexusModURL.FindStringSubmatch(summary); len(parts) == 2 {
		id, _ := strconv.Atoi(parts[1])
		if id > 0 {
			return refForNexusPage(ctx, m, id)
		}
	}
	if g := githubURL.FindStringSubmatch(summary); len(g) == 2 {
		repo := strings.TrimSuffix(g[1], ".git")
		return &Ref{Site: "GitHub", GitHub: repo, URL: "https://github.com/" + repo}
	}
	for _, id := range uniqueID.FindAllString(summary, -1) {
		if ref, ok := Locate(ctx, m, id, "", dependentKeys); ok && ref != nil && ref.URL != "" {
			return ref
		}
	}
	return nil
}

func refForNexusPage(ctx context.Context, m Meta, pageID int) *Ref {
	page, err := m.Page(ctx, pageID)
	if err != nil {
		return &Ref{Site: "Nexus", PageID: pageID, URL: nexus.ModURL(nexus.Game, pageID)}
	}
	r := meta.Ref{Site: "Nexus", ID: pageID}
	var best *Ref
	var bestTop string
	for _, f := range page.Downloads {
		if !strings.EqualFold(f.Type, "main") {
			continue
		}
		for _, mod := range f.Mods {
			cand, top := fileIn(page, r, mod.UniqueID, "")
			if cand == nil {
				continue
			}
			if c, isOrdered := meta.CompareVersions(top, bestTop); best == nil || (isOrdered && c > 0) {
				best, bestTop = cand, top
			}
		}
	}
	if best != nil {
		return best
	}
	return &Ref{Site: "Nexus", PageID: pageID, PageName: page.Name, URL: cmp.Or(page.PageURL, siteURL(r))}
}
