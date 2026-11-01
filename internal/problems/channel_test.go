package problems

import (
	"testing"
)

func TestUpsertChannelOffer(t *testing.T) {
	t.Parallel()
	x := inst("k", "Me.A", "0.5", true)
	existing := func(unofficial bool) *UpdatesResult {
		return &UpdatesResult{Updates: []Update{{Key: "k", ID: x.ModID(), Version: "1.0", URL: "old", NexusID: 1, Source: "SMAPI", Unofficial: unofficial}}}
	}
	cases := []struct {
		name       string
		r          *UpdatesResult
		version    string
		wantLen    int
		wantVer    string
		wantNexus  int
		wantSource string
	}{
		{"newer replaces", existing(false), "1.2", 1, "1.2", 7, "Nexus"},
		{"older is ignored", existing(false), "0.9", 1, "1.0", 1, "SMAPI"},
		{"equal is ignored", existing(false), "1.0", 1, "1.0", 1, "SMAPI"},
		{"unofficial entry is left and the offer appended", existing(true), "1.2", 2, "1.0", 1, "SMAPI"},
		{"nothing listed appends", &UpdatesResult{}, "1.2", 1, "1.2", 7, "Nexus"},
	}
	for _, c := range cases {
		upsertChannelOffer(c.r, x, c.version, "page", 7)
		if len(c.r.Updates) != c.wantLen {
			t.Errorf("%s: %d updates, want %d", c.name, len(c.r.Updates), c.wantLen)
			continue
		}
		u := c.r.Updates[0]
		if u.Version != c.wantVer || u.NexusID != c.wantNexus || u.Source != c.wantSource {
			t.Errorf("%s: first update = %+v", c.name, u)
		}
	}
	appended := &UpdatesResult{}
	upsertChannelOffer(appended, x, "1.2", "page", 7)
	if u := appended.Updates[0]; u.Installed != "0.5" || u.URL != "page" || u.Key != "k" {
		t.Errorf("appended offer = %+v", u)
	}
}
