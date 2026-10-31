package devsvc

import (
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// maxSiteID keeps an id inside what a site's own ids reach and what a JavaScript number holds.
const maxSiteID = 1_999_999_999

// NexusIDs name the mod page and the file a release replaces on Nexus Mods. FileID alone is enough to upload; ModID
// adds the changelog entry and the link to the page.
type NexusIDs struct {
	ModID  int `json:"modId,omitempty"`
	FileID int `json:"fileId,omitempty"`
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

func (e ProjectEdit) checkSites() error {
	for _, id := range []int{e.Nexus.ModID, e.Nexus.FileID, e.CurseForge.ProjectID} {
		if id < 0 || id > maxSiteID {
			return usererr.Wrap(usererr.Invalid, errors.New("site ids are whole numbers"))
		}
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
