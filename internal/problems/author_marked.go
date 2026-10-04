package problems

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

var (
	authorStatusWord = regexp.MustCompile(`(?i)\b(obsolete|deprecated|depreciated)\b`)
	nexusModLink     = regexp.MustCompile(`(?i)nexusmods\.com/stardewvalley/mods/([0-9]+)`)
)

func authorMarkedMods(home string, enabled []Installed) []Broken {
	var out []Broken
	for _, mod := range enabled {
		if pageID, fileID, ok := store.NexusFile(mod.Key); ok {
			details, ok := readCachedNexusDetails(home, pageID)
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
			if marked := markedBroken(mod, details.Page.Name, details.Page.Summary, details.Page.Description, file); marked != nil {
				out = append(out, *marked)
				continue
			}
		}
		if marked := markedManifest(mod); marked != nil {
			out = append(out, *marked)
		}
	}
	return out
}

// readCachedNexusDetails reads the details Nexus pages left in the cache, never fetching.
func readCachedNexusDetails(home string, pageID int) (nexussvc.Details, bool) {
	return meta.Peek[nexussvc.Details](&meta.Client{CacheDir: filepath.Join(home, "cache")}, nexussvc.DetailsName(pageID))
}

func markedManifest(mod Installed) *Broken {
	if match := statusMatch(mod.Name, true); match != "" {
		return authorBroken(mod, match, "Its installed manifest name")
	}
	if match := statusMatch(mod.Description, false); match != "" {
		return authorBroken(mod, match, "Its installed manifest description")
	}
	return nil
}

func markedBroken(mod Installed, name, summary, description string, file *nexus.File) *Broken {
	if match := statusMatch(name, true); match != "" {
		return authorBroken(mod, match, "Its Nexus mod name")
	}
	if match := statusMatch(summary, false); match != "" {
		return authorBrokenText(mod, match, "Its Nexus page summary", summary)
	}
	if match := statusMatch(description, false); match != "" {
		return authorBrokenText(mod, match, "Its Nexus page description", description)
	}
	if file != nil {
		if match := statusMatch(file.Name, true); match != "" {
			return authorBroken(mod, match, "Its installed file name")
		}
		if match := statusMatch(file.Description, false); match != "" {
			return authorBrokenText(mod, match, "Its installed file description", file.Description)
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

func authorBroken(mod Installed, quote, where string) *Broken {
	return authorBrokenWithReplacement(mod, quote, where)
}

func authorBrokenText(mod Installed, quote, where, text string) *Broken {
	b := authorBrokenWithReplacement(mod, quote, where)
	if ids := nexusModLink.FindStringSubmatch(text); len(ids) == 2 {
		b.Replacement = &Ref{Site: "Nexus", PageID: mustInt(ids[1]), URL: nexus.ModURL(nexus.Game, mustInt(ids[1]))}
	}
	return b
}

func authorBrokenWithReplacement(mod Installed, quote, where string) *Broken {
	status := "obsolete"
	if match := authorStatusWord.FindString(quote); match != "" {
		status = strings.ToLower(match)
		if status == "depreciated" {
			status = "deprecated"
		}
	}
	b := &Broken{
		Key: mod.Key, UniqueID: mod.UniqueID, Name: mod.Name, Status: status,
		Summary: fmt.Sprintf("%s says it is %s: %q", where, status, quote),
	}
	if ids := nexusModLink.FindStringSubmatch(quote); len(ids) == 2 {
		b.Replacement = &Ref{Site: "Nexus", PageID: mustInt(ids[1]), URL: nexus.ModURL(nexus.Game, mustInt(ids[1]))}
	}
	return b
}

func mustInt(value string) int {
	id, _ := strconv.Atoi(value)
	return id
}
