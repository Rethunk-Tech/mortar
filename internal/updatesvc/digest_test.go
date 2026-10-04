package updatesvc

import (
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestDecideUpdateDigestSameSetNoToast(t *testing.T) {
	prev := []string{"a\x00" + "1.0", "b\x00" + "2.0"}
	next := []string{"a\x00" + "1.0", "b\x00" + "2.0"}
	notify, persist := DecideUpdateDigest(settings.UpdateDigestEach, prev, next, "", time.Now())
	if notify || !slices.Equal(persist, prev) {
		t.Fatalf("same set: notify=%t persist=%v", notify, persist)
	}
}

func TestDecideUpdateDigestNewVersionToasts(t *testing.T) {
	prev := []string{"a\x00" + "1.0"}
	next := []string{"a\x00" + "2.0"}
	notify, persist := DecideUpdateDigest(settings.UpdateDigestEach, prev, next, "", time.Now())
	if !notify || !slices.Equal(persist, next) {
		t.Fatalf("new version: notify=%t persist=%v", notify, persist)
	}
}

func TestDecideUpdateDigestDailyThrottle(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	lastAt := now.Add(-2 * time.Hour).Format(time.RFC3339)
	prev := []string{"a\x00" + "1.0"}
	next := []string{"a\x00" + "2.0"}
	notify, persist := DecideUpdateDigest(settings.UpdateDigestDaily, prev, next, lastAt, now)
	if notify || !slices.Equal(persist, prev) {
		t.Fatalf("throttled: notify=%t persist=%v", notify, persist)
	}
	notify, _ = DecideUpdateDigest(settings.UpdateDigestDaily, prev, next, now.Add(-25*time.Hour).Format(time.RFC3339), now)
	if !notify {
		t.Fatal("expected toast after 24h")
	}
}

func TestDecideUpdateDigestOffPersistsWithoutToast(t *testing.T) {
	prev := []string{"a\x00" + "1.0"}
	next := []string{"a\x00" + "2.0"}
	notify, persist := DecideUpdateDigest(settings.UpdateDigestOff, prev, next, "", time.Now())
	if notify || !slices.Equal(persist, next) {
		t.Fatalf("off: notify=%t persist=%v", notify, persist)
	}
}

func TestPickDigestTargetTieOpensProfiles(t *testing.T) {
	profiles := []ProfileModUpdates{
		{Game: "stardew", ProfileID: "a", Updates: []problems.Update{{Key: "x", Version: "2"}}},
		{Game: "stardew", ProfileID: "b", Updates: []problems.Update{{Key: "y", Version: "2"}}},
	}
	_, notice := DigestFromProfiles(profiles)
	if !notice.OpenProfiles || notice.ProfileID != "" || notice.TotalUpdates != 2 || notice.ProfilesWith != 2 {
		t.Fatalf("tie: %#v", notice)
	}
}
