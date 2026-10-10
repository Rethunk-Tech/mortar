package folder_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/deploy"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// TestPlaceAndPurgeAThreeThousandFileProfile times the real launch path (lay the packages out in the profile, plan,
// place into the mods role folder, purge) for a profile of 300 per-file entries and one kept-whole script archive of
// 2,700 files. It is opt-in: MORTAR_SCALE=1. MORTAR_SCALE_MODS names the folder to place into, so a synced Documents
// folder can be timed, and MORTAR_SCALE_KB the size of each package (one byte without it).
func TestPlaceAndPurgeAThreeThousandFileProfile(t *testing.T) {
	if os.Getenv("MORTAR_SCALE") != "1" {
		t.Skip("set MORTAR_SCALE=1 to time a 3,000-file profile")
	}
	useFolderGame(t)
	dir := t.TempDir()
	ps := profile.OpenIn(dir, store.OpenAt(filepath.Join(dir, "store")))
	p, err := ps.Create(gameID, "Big")
	if err != nil {
		t.Fatal(err)
	}
	for i := range 300 {
		zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "s.zip"), map[string]string{fmt.Sprintf("s%03d.package", i): fmt.Sprint(i)})
		if _, err := ps.InstallSource(t.Context(), gameID, p.ID, zip, profile.Source{Kind: profile.KindCurseForge, Name: fmt.Sprint("Single", i), ModID: i + 1, FileID: 1}); err != nil {
			t.Fatal(err)
		}
	}
	body := "x"
	if kb, _ := strconv.Atoi(os.Getenv("MORTAR_SCALE_KB")); kb > 0 {
		body = strings.Repeat("x", kb<<10)
	}
	big := map[string]string{"scripts/m.ts4script": "s"}
	for i := range 2700 {
		big[fmt.Sprintf("w/%02d/b%04d.package", i/100, i)] = body
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "big.zip"), big)
	res, err := ps.InstallSource(t.Context(), gameID, p.ID, zip, profile.Source{Kind: profile.KindCurseForge, Name: "Whole", ModID: 999, FileID: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("entries = %d", len(res.Profile.Entries))

	measure := func(name string, fn func()) {
		w, c := time.Now(), cpu()
		fn()
		t.Logf("%-8s wall %v  cpu %v", name, time.Since(w).Round(time.Millisecond), (cpu() - c).Round(time.Millisecond))
	}
	profDir, _ := ps.ProfileDir(gameID, p.ID)
	mods := filepath.Join(t.TempDir(), "Mods")
	if at := os.Getenv("MORTAR_SCALE_MODS"); at != "" {
		mods = at
	}
	d, _ := deploy.Get("copy-into-install")
	l, _ := loader.Get("folder")
	plan := launchplan.New(launchplan.ModeProfile)
	var m deploy.Manifest
	measure("sync", func() {
		if err := ps.SyncPackages(gameID, p.ID); err != nil {
			t.Fatal(err)
		}
	})
	measure("plan", func() {
		if err := l.Contribute(t.Context(), plan, loader.ProfileView{Game: gameID, Dir: profDir}); err != nil {
			t.Fatal(err)
		}
	})
	t.Logf("plan files = %d", len(plan.Files))
	measure("place", func() {
		dp, err := d.Plan(deploy.View{JournalDir: filepath.Join(dir, "journal"), Roots: map[string]string{"mods": mods}}, filepath.Join(dir, "install"), plan.Files)
		if err != nil {
			t.Fatal(err)
		}
		if m, err = d.Apply(t.Context(), dp); err != nil {
			t.Fatal(err)
		}
	})
	measure("purge", func() {
		if err := d.Purge(t.Context(), m); err != nil {
			t.Fatal(err)
		}
	})
}
