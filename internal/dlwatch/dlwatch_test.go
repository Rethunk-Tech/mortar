package dlwatch

import (
	"testing"
)

func TestParseNexusFilename(t *testing.T) {
	inf, ok := ParseNexusFilename("ContentPatcher-1915-2-8-2-1700000000.zip")
	if !ok {
		t.Fatal("expected nexus name")
	}
	if inf.ModID != 1915 || inf.Name != "ContentPatcher" {
		t.Fatalf("got %+v", inf)
	}
	inf, ok = ParseNexusFilename("A_Mod-Name-42-1-0-1700000001.7z")
	if !ok || inf.ModID != 42 || inf.Name != "A Mod-Name" {
		t.Fatalf("got %+v ok=%v", inf, ok)
	}
	if _, ok := ParseNexusFilename("random.zip"); ok {
		t.Fatal("plain zip is not a nexus name")
	}
}
