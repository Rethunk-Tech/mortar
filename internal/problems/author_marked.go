package problems

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

var authorStatusWord = regexp.MustCompile(`(?i)\b(obsolete|deprecated|depreciated)\b`)

func authorMarkedMods(home, domain string, enabled []framework.Mod) []Broken {
	var out []Broken
	for _, im := range enabled {
		if pageID, fileID, ok := store.NexusFile(im.Key); ok {
			details, ok := readCachedNexusDetails(home, domain, pageID)
			if !ok {
				continue
			}
			var file *nexus.File
			for i := range details.Files {
				if details.Files[i].FileID == fileID {
					file = &details.Files[i]
					break
				}
			}
			if marked := markedBroken(im, domain, details.Page.Name, details.Page.Summary, details.Page.Description, file); marked != nil {
				out = append(out, *marked)
				continue
			}
		}
		if marked := markedManifest(im, domain); marked != nil {
			out = append(out, *marked)
		}
	}
	return out
}

// readCachedNexusDetails reads the details Nexus pages left in the cache, never fetching.
func readCachedNexusDetails(home, domain string, pageID int) (nexussvc.Details, bool) {
	return meta.Peek[nexussvc.Details](&meta.Client{CacheDir: filepath.Join(home, "cache")}, nexussvc.DetailsName(domain, pageID))
}

func markedManifest(im framework.Mod, domain string) *Broken {
	if match := statusMatch(im.Name, true); match != "" {
		return authorBroken(im, domain, match, "Its installed manifest name")
	}
	if match := statusMatch(im.Description, false); match != "" {
		return authorBroken(im, domain, match, "Its installed manifest description")
	}
	return nil
}

func markedBroken(im framework.Mod, domain, name, summary, description string, file *nexus.File) *Broken {
	if match := statusMatch(name, true); match != "" {
		return authorBroken(im, domain, match, "Its Nexus mod name")
	}
	if match := statusMatch(summary, false); match != "" {
		return authorBrokenText(im, domain, match, "Its Nexus page summary", summary)
	}
	if match := statusMatch(description, false); match != "" {
		return authorBrokenText(im, domain, match, "Its Nexus page description", description)
	}
	if file != nil {
		if match := statusMatch(file.Name, true); match != "" {
			return authorBroken(im, domain, match, "Its installed file name")
		}
		if match := statusMatch(file.Description, false); match != "" {
			return authorBrokenText(im, domain, match, "Its installed file description", file.Description)
		}
	}
	return nil
}

func statusMatch(text string, name bool) string {
	match := authorStatusWord.FindStringIndex(text)
	if match == nil {
		return ""
	}
	if name || appliesToMod(text, match[0]) {
		return quoteAround(text, match[0], match[1])
	}
	return ""
}

func appliesToMod(text string, at int) bool {
	prefix := strings.TrimSpace(text[:at])
	if prefix == "" || strings.HasPrefix(strings.ToLower(prefix), "this mod") || strings.HasPrefix(strings.ToLower(prefix), "this file") {
		return true
	}
	lineStart := strings.LastIndexAny(text[:at], "\r\n")
	line := strings.TrimSpace(text[lineStart+1:])
	return strings.HasPrefix(line, "#") || (line != "" && strings.ToUpper(line) == line && !strings.ContainsAny(line, ".!?"))
}

func quoteAround(text string, start, end int) string {
	begin := strings.LastIndexAny(text[:start], ".!?\r\n")
	if begin < 0 {
		begin = 0
	} else {
		begin++
	}
	finish := strings.IndexAny(text[end:], ".!?\r\n")
	if finish < 0 {
		finish = len(text) - end
	}
	finish += end
	sentence := strings.TrimSpace(text[begin:finish])
	if len([]rune(sentence)) > 80 {
		sentence = string([]rune(sentence)[:77]) + "..."
	}
	return sentence
}

func authorBroken(im framework.Mod, domain, quote, where string) *Broken {
	return authorBrokenWithReplacement(im, domain, quote, where)
}

func authorBrokenText(im framework.Mod, domain, quote, where, text string) *Broken {
	b := authorBrokenWithReplacement(im, domain, quote, where)
	if ids := modPageIDs(text, domain); len(ids) > 0 {
		b.Replacement = &Ref{Site: "Nexus", PageID: ids[0], URL: nexus.ModURL(domain, ids[0])}
	}
	return b
}

func authorBrokenWithReplacement(im framework.Mod, domain, quote, where string) *Broken {
	status := "obsolete"
	if match := authorStatusWord.FindString(quote); match != "" {
		status = strings.ToLower(match)
		if status == "depreciated" {
			status = "deprecated"
		}
	}
	b := &Broken{
		Key: im.Key, ID: im.ModID(), Name: im.Name, Status: status,
		Summary: fmt.Sprintf("%s says it is %s: %q", where, status, quote),
	}
	if ids := modPageIDs(quote, domain); len(ids) > 0 {
		b.Replacement = &Ref{Site: "Nexus", PageID: ids[0], URL: nexus.ModURL(domain, ids[0])}
	}
	return b
}
