package profile

import (
	"strconv"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestSourceKey(t *testing.T) {
	tests := []struct {
		name string
		src  Source
		key  string
		ok   bool
	}{
		{
			name: "nexus",
			src:  Source{Kind: KindNexus, ModID: 10, FileID: 20},
			key:  store.NexusKey(10, 20),
			ok:   true,
		},
		{
			name: "thunderstore",
			src:  Source{Kind: KindThunderstore, Name: "Author-Mod", Version: "1.2.3"},
			key:  store.PackageKey("Author-Mod", "1.2.3"),
			ok:   true,
		},
		{
			name: "curseforge",
			src:  Source{Kind: KindCurseForge, Name: "pack", FileID: 99},
			key:  store.PackageKey(KindCurseForge+":pack", strconv.Itoa(99)),
			ok:   true,
		},
		{
			name: "modio",
			src:  Source{Kind: KindModio, Name: "pack", FileID: 7},
			key:  store.PackageKey(KindModio+":pack", strconv.Itoa(7)),
			ok:   true,
		},
		{
			name: "modrinth",
			src:  Source{Kind: KindModrinth, Name: "slug", Version: "2.0.0"},
			key:  store.PackageKey(KindModrinth+":slug", "2.0.0"),
			ok:   true,
		},
		{
			name: "itch",
			src:  Source{Kind: KindItch, Name: "user/game", Version: "1.0"},
			key:  store.PackageKey(KindItch+":user/game", "1.0"),
			ok:   true,
		},
		{
			name: "vsmoddb",
			src:  Source{Kind: KindVSModDB, Name: "vsmod", Version: "1.4.0"},
			key:  store.PackageKey(KindVSModDB+":vsmod", "1.4.0"),
			ok:   true,
		},
		{
			name: "paradox",
			src:  Source{Kind: KindParadox, Name: "pdox", Version: "3.1"},
			key:  store.PackageKey(KindParadox+":pdox", "3.1"),
			ok:   true,
		},
		{
			name: "local",
			src:  Source{Kind: KindLocal, Name: "file.zip"},
		},
		{
			name: "github",
			src:  Source{Kind: KindGitHub, Repo: "o/r", Tag: "v1", Asset: "a.zip"},
		},
		{
			name: "unknown",
			src:  Source{Kind: "nope"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := SourceKey(tt.src)
			if ok != tt.ok || got != tt.key {
				t.Fatalf("SourceKey(%+v) = %q, %v; want %q, %v", tt.src, got, ok, tt.key, tt.ok)
			}
		})
	}
}
