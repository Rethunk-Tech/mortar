package devsvc

import (
	"errors"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// maxSiteID keeps an id inside what a site's own ids reach and what a JavaScript number holds.
const maxSiteID = 1_999_999_999

// NexusIDs name the mod page and the file a release replaces on Nexus Mods. Nexus's v3 API types every id as a string
// (a mod's can be a composite uid), so they are kept as typed. FileID alone is enough to upload; ModID adds the
// changelog entry and the link to the page.
type NexusIDs struct {
	ModID  string `json:"modId,omitempty"`
	FileID string `json:"fileId,omitempty"`
}

// CurseForgeIDs name the project a release is uploaded to on CurseForge.
type CurseForgeIDs struct {
	ProjectID int `json:"projectId,omitempty"`
}

// siteIDs are the per-site ids a project's record keeps, so Release can offer the sites the project is on.
type siteIDs struct {
	Nexus      NexusIDs      `json:"nexus,omitzero"`
	CurseForge CurseForgeIDs `json:"curseforge,omitzero"`
}

// nexusID is what an id on Nexus Mods may hold, which also keeps it a single URL path segment.
var nexusID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func (e ProjectEdit) checkSites() error {
	for _, id := range []string{e.Nexus.ModID, e.Nexus.FileID} {
		if id != "" && !nexusID.MatchString(id) {
			return usererr.Wrap(usererr.Invalid, errors.New("a Nexus Mods id holds letters, digits, dashes and underscores"))
		}
	}
	if id := e.CurseForge.ProjectID; id < 0 || id > maxSiteID {
		return usererr.Wrap(usererr.Invalid, errors.New("the CurseForge project id is a whole number"))
	}
	return nil
}

// readSites is the site ids of the project in dir, from the record it keeps.
func readSites(dir string) siteIDs {
	if rec, ok := readRawRecord(dir); ok {
		return rec.siteIDs
	}
	rec, _ := readTemplateRecord(dir)
	return rec.siteIDs
}

// writeSites records the site ids of the project in dir, in the record it keeps.
func writeSites(dir string, ids siteIDs) error {
	if rec, ok := readRawRecord(dir); ok {
		rec.siteIDs = ids
		return writeRecord(dir, rec)
	}
	rec, ok := readTemplateRecord(dir)
	if !ok && ids == (siteIDs{}) {
		return nil
	}
	rec.siteIDs = ids
	return writeTemplateRecord(dir, rec)
}

// changelogOf is the notes under "## version" in a CHANGELOG.md, without the heading.
func changelogOf(changelog, version string) string {
	var out []string
	in := false
	for line := range strings.Lines(changelog) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if in {
				break
			}
			in = trimmed == "## "+version
			continue
		}
		if in {
			out = append(out, strings.TrimRight(line, "\r\n"))
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
