package problems

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func applyChannelFileOffers(ctx context.Context, m Meta, domain string, asked []Installed, r *UpdatesResult) {
	for _, x := range asked {
		ch := profile.NormalizeChannel(x.UpdateChannel)
		if ch == profile.ChannelMain {
			continue
		}
		modID := nexusID(x.UpdateKeys)
		if modID == 0 {
			if pageID, _, ok := store.NexusFile(x.Key); ok {
				modID = pageID
			}
		}
		if modID == 0 {
			continue
		}
		page, err := m.Page(ctx, modID)
		if err != nil {
			continue
		}
		files := make([]nexus.File, 0, len(page.Downloads))
		for _, d := range page.Downloads {
			files = append(files, nexus.File{
				FileID: int(d.ID), FileName: d.FileName, Name: d.FileName, Version: d.Version, Category: d.Type,
			})
		}
		f, ok := profile.NewestChannelFile(ch, x.Version, files)
		if !ok || downloaded(x, f.Version) {
			continue
		}
		upsertChannelOffer(r, x, f.Version, siteURL(domain, meta.Ref{Site: "Nexus", ID: modID}), modID)
	}
}

func upsertChannelOffer(r *UpdatesResult, x Installed, version, pageURL string, nexusID int) {
	for i, u := range r.Updates {
		if u.Key != x.Key || u.ID != x.ModID() || u.Unofficial {
			continue
		}
		c, ok := meta.CompareVersions(version, u.Version)
		if !ok || c <= 0 {
			return
		}
		r.Updates[i].Version = version
		r.Updates[i].URL = pageURL
		r.Updates[i].NexusID = nexusID
		r.Updates[i].Source = "Nexus"
		return
	}
	r.Updates = append(r.Updates, Update{
		Key: x.Key, ID: x.ModID(), Name: x.Name, Installed: x.Version,
		Version: version, URL: pageURL, NexusID: nexusID, Source: "Nexus",
	})
}

func keepPrerelease(m Installed, includePrerelease bool, version, installed string) bool {
	if includePrerelease || profile.NormalizeChannel(m.UpdateChannel) == profile.ChannelBeta {
		return true
	}
	return !hasPrerelease(version) || hasPrerelease(installed)
}
