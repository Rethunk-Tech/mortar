package updatesvc

import (
	"slices"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

const (
	ModUpdateDigestEvent = "updates:digest"
	modUpdateDigestDay   = 24 * time.Hour
)

// ModUpdateDigestNotice is sent to the UI when a background check finds a new digest.
type ModUpdateDigestNotice struct {
	Game         string `json:"game"`
	ProfileID    string `json:"profile"`
	OpenProfiles bool   `json:"openProfiles"`
	TotalUpdates int    `json:"totalUpdates"`
	ProfilesWith int    `json:"profilesWith"`
}

// ProfileModUpdates is one profile's official update list from a background scan.
type ProfileModUpdates struct {
	Game        string
	ProfileID   string
	ProfileName string
	Updates     []problems.Update
}

func digestKey(entryKey, version string) string {
	return entryKey + "\x00" + version
}

func digestKeysFromUpdates(updates []problems.Update) []string {
	seen := make(map[string]struct{}, len(updates))
	for _, u := range updates {
		if u.Unofficial {
			continue
		}
		k := digestKey(u.Key, u.Version)
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func mergeDigestKeys(profiles []ProfileModUpdates) []string {
	seen := make(map[string]struct{})
	for _, p := range profiles {
		for _, k := range digestKeysFromUpdates(p.Updates) {
			seen[k] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func digestShownWithinDay(at string, now time.Time) bool {
	if at == "" {
		return false
	}
	then, err := time.Parse(time.RFC3339, at)
	if err != nil {
		return false
	}
	return now.Sub(then) < modUpdateDigestDay
}

// DecideUpdateDigest reports whether to toast and the digest to persist after a background scan.
func DecideUpdateDigest(
	mode string,
	prev []string,
	next []string,
	lastAt string,
	now time.Time,
) (notify bool, persist []string) {
	if slices.Equal(prev, next) {
		return false, prev
	}
	switch mode {
	case settings.UpdateDigestOff:
		return false, next
	case settings.UpdateDigestEach:
		return true, next
	case settings.UpdateDigestDaily:
		// A held-back digest keeps the previous set, so tomorrow's check still announces these updates.
		if digestShownWithinDay(lastAt, now) {
			return false, prev
		}
		return true, next
	default:
		return false, next
	}
}

func countOfficialUpdates(updates []problems.Update) int {
	return len(digestKeysFromUpdates(updates))
}

func pickDigestTarget(profiles []ProfileModUpdates) ModUpdateDigestNotice {
	var best ProfileModUpdates
	bestCount := 0
	tied := 0
	profilesWith := 0
	total := 0
	for _, p := range profiles {
		n := countOfficialUpdates(p.Updates)
		if n == 0 {
			continue
		}
		profilesWith++
		total += n
		if n > bestCount {
			bestCount = n
			best = p
			tied = 1
			continue
		}
		if n == bestCount {
			tied++
		}
	}
	notice := ModUpdateDigestNotice{TotalUpdates: total, ProfilesWith: profilesWith}
	if profilesWith == 0 {
		return notice
	}
	if tied > 1 {
		notice.OpenProfiles = true
		notice.Game = best.Game
		return notice
	}
	notice.Game = best.Game
	notice.ProfileID = best.ProfileID
	return notice
}

// DigestFromProfiles builds the merged digest and summary for a scan.
func DigestFromProfiles(profiles []ProfileModUpdates) (keys []string, notice ModUpdateDigestNotice) {
	keys = mergeDigestKeys(profiles)
	notice = pickDigestTarget(profiles)
	return keys, notice
}
