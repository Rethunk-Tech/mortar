package modmenu

import (
	"slices"
	"testing"
)

func TestParseTarget(t *testing.T) {
	got, err := ParseTarget(`{"game":"stardew","profile":"p1","key":"k","uniqueId":"me.mod"}`)
	if err != nil || got != (Target{Game: "stardew", Profile: "p1", Key: "k", UniqueID: "me.mod"}) {
		t.Fatalf("got %+v, %v", got, err)
	}
	for _, bad := range []string{``, `not json`, `{"game":"stardew"}`, `{"game":"g","profile":"p","key":"k"}`} {
		if _, err := ParseTarget(bad); err == nil {
			t.Errorf("ParseTarget(%q) accepted", bad)
		}
	}
}

func TestActionsFollowState(t *testing.T) {
	full := Actions(State{Enabled: true, Host: HostNexus, Removable: true})
	if !slices.Equal(full, []string{Toggle, Details, Page, Files, Remove}) {
		t.Errorf("full: %v", full)
	}
	bare := Actions(State{})
	if !slices.Equal(bare, []string{Toggle, Details, Files}) {
		t.Errorf("bare: %v", bare)
	}
}

func TestEveryStateHasItsOwnMenuID(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range states() {
		seen[MenuID(s)] = true
	}
	if len(seen) != 12 {
		t.Errorf("%d distinct menu ids for 12 states", len(seen))
	}
	if id := MenuID(State{Enabled: true, Host: "", Removable: true}); id != "mod-menu-on-nopage-remove" {
		t.Errorf("id %q", id)
	}
}

func TestToggleLabelReflectsState(t *testing.T) {
	if label(english, Toggle, State{Enabled: true}) != "Disable" || label(english, Toggle, State{}) != "Enable" {
		t.Error("toggle label")
	}
}

func TestPageLabelNamesTheHost(t *testing.T) {
	l := Labels{OpenNexus: "Nexus", OpenGitHub: "GitHub"}
	if label(l, Page, State{Host: HostNexus}) != "Nexus" || label(l, Page, State{Host: HostGitHub}) != "GitHub" {
		t.Error("page label")
	}
}
