package problems

import (
	"cmp"
	"context"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

var (
	nexusModPath = regexp.MustCompile(`(?i)^([a-z0-9]+)/mods/(\d+)`)
	uniqueID     = regexp.MustCompile(`\b([A-Za-z][A-Za-z0-9_]*\.[A-Za-z][A-Za-z0-9_]*)\b`)
)

// linkParts splits each web address in text, with or without its scheme, into its lower-cased host (without a leading
// "www.") and the path after the first slash. An address is a run of text with no space, bracket or quote in it.
func linkParts(text string) (hosts, paths []string) {
	for _, tok := range strings.FieldsFunc(text, func(r rune) bool { return unicode.IsSpace(r) || strings.ContainsRune(`<>()[]"'`, r) }) {
		tok = strings.TrimPrefix(strings.TrimPrefix(tok, "https://"), "http://")
		host, path, ok := strings.Cut(tok, "/")
		if !ok {
			continue
		}
		hosts = append(hosts, strings.TrimPrefix(strings.ToLower(host), "www."))
		paths = append(paths, path)
	}
	return hosts, paths
}

// modPageIDs lists the mod ids of the Nexus page links in text that belong to domain.
func modPageIDs(text, domain string) []int {
	var ids []int
	hosts, paths := linkParts(text)
	for i, host := range hosts {
		if host != "nexusmods.com" {
			continue
		}
		parts := nexusModPath.FindStringSubmatch(paths[i])
		if parts == nil || !strings.EqualFold(parts[1], domain) {
			continue
		}
		if id, err := strconv.Atoi(parts[2]); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// githubLinkRepo is the "owner/repo" of the first GitHub repository link in text, or "".
func githubLinkRepo(text string) string {
	hosts, paths := linkParts(text)
	for i, host := range hosts {
		if host != "github.com" {
			continue
		}
		path, _, _ := strings.Cut(paths[i], "#")
		path, _, _ = strings.Cut(path, "?")
		owner, rest, ok := strings.Cut(path, "/")
		name, _, _ := strings.Cut(rest, "/")
		if ok && owner != "" && name != "" {
			return owner + "/" + name
		}
	}
	return ""
}

// replacementFromSummary picks a mod to install when SMAPI's compatibility summary names one.
func replacementFromSummary(ctx context.Context, m Meta, domain string, dependentKeys []string, summary string) *Ref {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil
	}
	if ids := modPageIDs(summary, domain); len(ids) > 0 && ids[0] > 0 {
		return refForNexusPage(ctx, m, domain, ids[0])
	}
	if g := githubLinkRepo(summary); g != "" {
		repo := strings.TrimSuffix(g, ".git")
		return &Ref{Site: "GitHub", GitHub: repo, URL: "https://github.com/" + repo}
	}
	for _, id := range uniqueID.FindAllString(summary, -1) {
		if ref, ok := Locate(ctx, m, domain, mod.SMAPI(id), "", dependentKeys); ok && ref != nil && ref.URL != "" {
			return ref
		}
	}
	return nil
}

func refForNexusPage(ctx context.Context, m Meta, domain string, pageID int) *Ref {
	page, err := m.Page(ctx, pageID)
	if err != nil {
		return &Ref{Site: "Nexus", PageID: pageID, URL: nexus.ModURL(domain, pageID)}
	}
	r := meta.Ref{Site: "Nexus", ID: pageID}
	var best *Ref
	var bestTop string
	for _, f := range page.Downloads {
		if !strings.EqualFold(f.Type, "main") {
			continue
		}
		for _, im := range f.Mods {
			cand, top := fileIn(page, domain, r, im.ModID(), "")
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
	return &Ref{Site: "Nexus", PageID: pageID, PageName: page.Name, URL: cmp.Or(page.PageURL, siteURL(domain, r))}
}
