package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/datasvc"
	"github.com/Rethunk-Tech/mortar/internal/devsvc"
	"github.com/Rethunk-Tech/mortar/internal/packsvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/templates"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// writeModZip writes a one-mod SMAPI archive, its manifest in folder.
func writeModZip(t *testing.T, folder, uniqueID, name string) string {
	t.Helper()
	manifest := fmt.Sprintf(`{"Name":%q,"UniqueID":%q,"Version":"1.0.0","MinimumApiVersion":"4.0.0"}`, name, uniqueID)
	return testfs.WriteZip(t, filepath.Join(t.TempDir(), uniqueID+".zip"), map[string]string{
		folder + "/manifest.json": manifest,
	})
}

func TestRemoveReportsEveryModOfAnEntryAndStopsAtTheFirstFailure(t *testing.T) {
	s := services(t)
	p, err := s.Profiles.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Store.InstallSource(t.Context(), "stardew", p.ID, writeModZip(t, "Mod", "Test.Alpha", "Alpha"),
		profile.Source{Kind: profile.KindNexus, Name: "alpha.zip", ModID: 11, FileID: 1, Version: "1.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Profile.Entries) != 1 {
		t.Fatalf("entries = %+v", res.Profile.Entries)
	}
	// One download can hold several mods; the reply names them all, not only the one named.
	shared := res.Profile
	shared.Entries = slices.Clone(shared.Entries)
	shared.Entries[0].Mods = append(slices.Clone(shared.Entries[0].Mods), profile.Component{ID: "smapi:Test.Beta", Name: "Beta"})
	key := shared.Entries[0].Key

	out, err := s.remove("stardew", p.ID, shared, []string{key, "no-such-entry"})
	if err == nil {
		t.Fatal("removing an entry the profile lacks must fail")
	}
	if !slices.Equal(out.Mods, []string{"Alpha", "Beta"}) {
		t.Fatalf("removed = %v, want both mods of the entry", out.Mods)
	}
	got, err := s.Profiles.Get("stardew", p.ID)
	if err != nil || len(got.Entries) != 0 {
		t.Fatalf("after remove: %+v, %v", got.Entries, err)
	}
}

func TestCopyProfileSkipsRenameWithoutANameAndKeepsTheCopyWhenRenameFails(t *testing.T) {
	s := services(t)
	p, err := s.Profiles.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	dup, err := s.copyProfile("stardew", p.ID, "")
	if err != nil || !strings.HasSuffix(dup.Name, " copy") {
		t.Fatalf("copy without a name = %+v, %v", dup, err)
	}
	if _, err := s.copyProfile("stardew", p.ID, "bad\nname"); err == nil || !strings.Contains(err.Error(), "control character") {
		t.Fatalf("a name that cannot be a profile name must fail the rename: %v", err)
	}
	all, err := s.Profiles.List("stardew")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("a failed rename must leave its duplicate: %v", profile.Names(all))
	}
}

func TestDataCleanupAndMovePreviewsLeaveTheDiskAlone(t *testing.T) {
	s, items := servicesWithItems(t)
	s.Data = datasvc.NewService(items, s.Store, nil)

	key := store.PackageKey("Ns-A", "1.0.0")
	files := t.TempDir()
	if err := os.WriteFile(filepath.Join(files, "manifest.json"), []byte(`{"name":"A","namespace":"Ns","version_number":"1.0.0","dependencies":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := items.AddDir(t.Context(), "stardew", key, files); err != nil {
		t.Fatal(err)
	}
	dir, err := items.Dir("stardew", key)
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.dataCleanup(Params{Preview: true})
	if err != nil {
		t.Fatal(err)
	}
	if pv, ok := res.(datasvc.Preview); !ok || len(pv.Items) == 0 {
		t.Fatalf("cleanup preview = %#v, want the unreferenced store item", res)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("a cleanup preview removed the item: %v", err)
	}
	if _, err := s.dataCleanup(Params{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err == nil {
		t.Fatal("the cleanup kept the unreferenced store item")
	}

	before, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "moved")
	res, err = s.dataMove(Params{Preview: true, Path: dest})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.(datasvc.MoveEstimate); !ok {
		t.Fatalf("move preview = %#v", res)
	}
	if _, err := os.Stat(dest); err == nil {
		t.Fatal("a move preview created the destination")
	}
	if after, err := datadir.Dir(); err != nil || after != before {
		t.Fatalf("a move preview moved the data folder to %q", after)
	}
}

func TestPackAndBackupVerbsRefuseAMissingPathOrProfile(t *testing.T) {
	s := services(t)
	s.Packs = &packsvc.Service{}
	for _, tc := range []struct {
		name string
		call func() (any, error)
		want string
	}{
		{"modpack without a path", func() (any, error) { return s.packExportModpack(Params{}) }, "name the modpack zip to write"},
		{"r2z without a path", func() (any, error) { return s.packExportR2z(Params{}) }, "name the .r2z file to write"},
		{"backup without a path", func() (any, error) { return s.profileBackup(Params{}) }, "name the backup file to write"},
		{"modpack without a profile", func() (any, error) { return s.packExportModpack(Params{Path: "x.zip"}) }, "name a profile"},
		{"r2z without a profile", func() (any, error) { return s.packExportR2z(Params{Path: "x.r2z"}) }, "name a profile"},
		{"backup without a profile", func() (any, error) { return s.profileBackup(Params{Path: "x.backup"}) }, "name a profile"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.call(); err == nil || err.Error() != tc.want {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
	if _, err := (&Services{}).packExportModpack(Params{}); err == nil || err.Error() != "pack export is unavailable" {
		t.Fatalf("without the pack service: %v", err)
	}
	if _, err := (&Services{}).profileRestore(t.Context(), Params{}); err == nil || err.Error() != "profile restore is unavailable" {
		t.Fatalf("without the pack service: %v", err)
	}
}

func TestDevVerbsNeedDevelopmentAndRemoveByProjectName(t *testing.T) {
	s := services(t)
	if _, err := s.devClone(t.Context(), Params{Game: "stardew"}); !errors.Is(err, errNoDev) {
		t.Fatalf("clone without development: %v", err)
	}
	if _, err := s.devClone(t.Context(), Params{Game: "stardew", Profile: "nope"}); err == nil {
		t.Fatal("clone must resolve the profile it names")
	}
	if _, err := s.devRemove(Params{Game: "stardew", ID: "MyMod"}); !errors.Is(err, errNoDev) {
		t.Fatalf("remove without development: %v", err)
	}

	s.SettingsSvc = settings.NewService(s.Settings)
	s.Dev = &devsvc.Service{Settings: s.SettingsSvc}
	dir := filepath.Join(t.TempDir(), "mymod")
	if err := s.SettingsSvc.AddDevProject("stardew", settings.DevProject{Dir: dir, Name: "MyMod"}); err != nil {
		t.Fatal(err)
	}
	got, err := s.devRemove(Params{Game: "stardew", ID: "mymod"})
	if err != nil || got.Dir != dir {
		t.Fatalf("remove by name = %+v, %v", got, err)
	}
	if left := s.Dev.Projects("stardew"); len(left) != 0 {
		t.Fatalf("the removed project stayed: %+v", left)
	}
	if _, err := s.devRemove(Params{Game: "stardew", ID: "mymod"}); usererr.KindOf(err) != usererr.NotFound {
		t.Fatalf("an unknown project must be not found, got %v", err)
	}
}

func TestProfileBackupThenRestoreMakesANewProfile(t *testing.T) {
	s := services(t)
	s.Packs = &packsvc.Service{Profiles: packProfiles{s.Profiles, s.Store}}
	p, err := s.Profiles.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "farm.backup")
	out, err := s.profileBackup(Params{Game: "stardew", Profile: p.ID, Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if m, ok := out.(map[string]string); !ok || m["path"] != path {
		t.Fatalf("backup = %#v", out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	var events []string
	s.Emit = func(name string, _ any) { events = append(events, name) }
	if _, err := s.profileRestore(t.Context(), Params{Game: "stardew", Path: path}); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(events, ChangedEvent) {
		t.Fatalf("a restore must tell the window, got %v", events)
	}
	if all, err := s.Profiles.List("stardew"); err != nil || len(all) != 2 {
		t.Fatalf("restore made %d profiles, %v", len(all), err)
	}
}

// packProfiles is the profile store a pack service reads, which spans the service and its store.
type packProfiles struct {
	*profile.Service
	store *profile.Store
}

func (p packProfiles) ProfileDir(game, id string) (string, error) {
	return p.store.ProfileDir(game, id)
}

func (p packProfiles) Backup(game, id string) (profile.Profile, map[string][]byte, error) {
	return p.store.Backup(game, id)
}

func (p packProfiles) BackupDirs(game string, prof profile.Profile, keep func(profile.Entry) bool) (map[string]string, error) {
	return p.store.BackupDirs(game, prof, keep)
}

func (p packProfiles) RestoreBackup(ctx context.Context, game string, prof profile.Profile, files map[string][]byte, dirs map[string]string) (profile.Profile, []profile.Entry, error) {
	return p.store.RestoreBackup(ctx, game, prof, files, dirs)
}

func (p packProfiles) SetModpackDraft(game, id string, d profile.ModpackDraft) (profile.Profile, error) {
	return p.store.SetModpackDraft(game, id, d)
}

func TestTemplatesWriteNeedsItsServiceAndRefusesUnreadableBodies(t *testing.T) {
	if _, err := (&Services{}).templatesWrite("templates.apply", Params{}); !errors.Is(err, errUnavailable) {
		t.Fatalf("without the template service: %v", err)
	}
	s := &Services{Templates: new(templates.Service)}
	if _, err := s.templatesWrite("templates.undo", Params{Body: json.RawMessage("{")}); err == nil || !strings.HasPrefix(err.Error(), "undo data:") {
		t.Fatalf("an unreadable undo = %v", err)
	}
	if _, err := s.templatesWrite("templates.restore", Params{Body: json.RawMessage("{")}); err == nil || !strings.HasPrefix(err.Error(), "template data:") {
		t.Fatalf("an unreadable template = %v", err)
	}
}

func TestApplyEverywhereWithoutUpdatesReportsNothing(t *testing.T) {
	s := services(t)
	if _, err := s.Profiles.Create("stardew", "Farm"); err != nil {
		t.Fatal(err)
	}
	got, err := s.applyEverywhere(t.Context(), Params{Game: "stardew"})
	if err != nil || len(got.Updated) != 0 || len(got.Skipped) != 0 {
		t.Fatalf("apply everywhere = %+v, %v", got, err)
	}
}
