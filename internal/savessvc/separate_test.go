package savessvc

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/backup"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// separateEnv is a service with the shared saves holding Farm_1, a profile "own" keeping its saves separate with
// Farm_2, and a profile "shared" that does not.
func separateEnv(t *testing.T) (s *Service, shared, ownDir string, own, plain profile.Profile) {
	t.Helper()
	s, shared = backupService(t)
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	if s.profiles, err = profile.Open(items); err != nil {
		t.Fatal(err)
	}
	if own, err = s.profiles.Create("stardew", "Own"); err != nil {
		t.Fatal(err)
	}
	if own, err = s.profiles.SetSeparateSaves("stardew", own.ID, true, false); err != nil {
		t.Fatal(err)
	}
	if plain, err = s.profiles.Create("stardew", "Shared"); err != nil {
		t.Fatal(err)
	}
	if ownDir, err = s.profiles.SavesFolder("stardew", own.ID); err != nil {
		t.Fatal(err)
	}
	writeFarm(t, shared, "Farm_1", "Shared", "shared-v1")
	writeFarm(t, ownDir, "Farm_2", "Own", "own-v1")
	return s, shared, ownDir, own, plain
}

func TestSeparateSavesListOnlyTheProfilesOwn(t *testing.T) {
	s, _, _, own, plain := separateEnv(t)
	for _, c := range []struct {
		profile string
		want    string
	}{{own.ID, "Farm_2"}, {plain.ID, "Farm_1"}} {
		sc, err := s.scannerFor("stardew", c.profile)
		if err != nil {
			t.Fatal(err)
		}
		infos, err := sc.Scan(nil)
		if err != nil || len(infos) != 1 || infos[0].Folder != c.want {
			t.Fatalf("profile %s lists %+v, %v; want only %s", c.profile, infos, err, c.want)
		}
	}
}

func TestSeparateSavesBackupRestoreAndOpenActOnTheProfilesOwn(t *testing.T) {
	s, shared, ownDir, own, plain := separateEnv(t)
	if made, err := s.CreateBackup("stardew", own.ID, "Farm_1"); made || err != nil {
		t.Fatalf("backed up the shared Farm_1 for the separate profile: %v, %v", made, err)
	}
	if made, err := s.CreateBackup("stardew", own.ID, "Farm_2"); !made || err != nil {
		t.Fatalf("backup of own Farm_2 = %v, %v", made, err)
	}
	if made, err := s.CreateBackup("stardew", plain.ID, "Farm_2"); made || err != nil {
		t.Fatalf("backed up the separate Farm_2 for the shared profile: %v, %v", made, err)
	}
	if made, err := s.CreateBackup("stardew", plain.ID, "Farm_1"); !made || err != nil {
		t.Fatalf("backup of shared Farm_1 = %v, %v", made, err)
	}

	listed, err := s.ListBackups("stardew", "")
	if err != nil || len(listed) != 2 {
		t.Fatalf("list = %+v, %v", listed, err)
	}
	byFolder := map[string]string{}
	for _, b := range listed {
		byFolder[b.Saves[0].Folder] = b.Name
	}
	write := func(dir, folder, body string) {
		if err := fsx.WriteFile(filepath.Join(dir, folder, folder), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	read := func(dir, folder string) string {
		got, err := fsx.ReadFile(filepath.Join(dir, folder, folder))
		if err != nil {
			t.Fatal(err)
		}
		return string(got)
	}
	write(ownDir, "Farm_2", "own-v2")
	write(shared, "Farm_1", "shared-v2")
	if err := s.RestoreBackup("stardew", own.ID, byFolder["Farm_2"], nil); err != nil {
		t.Fatal(err)
	}
	if got := read(ownDir, "Farm_2"); got != "own-v1" {
		t.Fatalf("own Farm_2 = %q after restore", got)
	}
	if fsx.IsDir(filepath.Join(shared, "Farm_2")) {
		t.Fatal("restore for the separate profile wrote into the shared saves")
	}
	if err := s.RestoreBackup("stardew", plain.ID, byFolder["Farm_1"], nil); err != nil {
		t.Fatal(err)
	}
	if got := read(shared, "Farm_1"); got != "shared-v1" {
		t.Fatalf("shared Farm_1 = %q after restore", got)
	}
	if fsx.IsDir(filepath.Join(ownDir, "Farm_1")) {
		t.Fatal("restore for the shared profile wrote into the separate saves")
	}

	for _, c := range []struct {
		profile, folder, want string
	}{{own.ID, "Farm_2", filepath.Join(ownDir, "Farm_2")}, {plain.ID, "Farm_1", filepath.Join(shared, "Farm_1")}} {
		if got, err := s.saveFolderPath("stardew", c.profile, c.folder); err != nil || got != c.want {
			t.Fatalf("open %s for %s = %q, %v; want %q", c.folder, c.profile, got, err, c.want)
		}
	}
	for _, c := range [][2]string{{own.ID, "Farm_1"}, {plain.ID, "Farm_2"}} {
		if _, err := s.saveFolderPath("stardew", c[0], c[1]); err == nil {
			t.Fatalf("open %s for %s found a save the profile does not read", c[1], c[0])
		}
	}
	if slices.Contains([]string{byFolder["Farm_1"], byFolder["Farm_2"]}, "") {
		t.Fatalf("backups by folder %v", byFolder)
	}
}

func TestScheduledBackupsCoverEachSeparateSavesFolderOnce(t *testing.T) {
	s, _, ownDir, own, _ := separateEnv(t)
	writeFarm(t, ownDir, "Farm_1", "Own", "own-v1")
	if _, err := s.settings.Update(func(v *settings.Settings) {
		if err := settings.ApplyKeyGame(v, "saveBackupHours", "6", "stardew"); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	scheduled := func() []string {
		listed, err := s.ListBackups("stardew", "")
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, b := range listed {
			if b.Kind == backup.KindScheduled {
				out = append(out, b.Profile+":"+b.Saves[0].Folder)
			}
		}
		slices.Sort(out)
		return out
	}
	now := time.Now()
	s.scheduledTickFor("stardew", now)
	want := []string{":Farm_1", own.ID + ":Farm_1", own.ID + ":Farm_2"}
	slices.Sort(want)
	if got := scheduled(); !slices.Equal(got, want) {
		t.Fatalf("scheduled = %v, want %v", got, want)
	}
	s.scheduledTickFor("stardew", now.Add(7*time.Hour))
	if got := scheduled(); !slices.Equal(got, want) {
		t.Fatalf("unchanged saves backed up again: %v", got)
	}
}

func TestAProfilesOwnSavesFollowItsOwnBackupSchedule(t *testing.T) {
	s, _, ownDir, own, _ := separateEnv(t)
	writeFarm(t, ownDir, "Farm_1", "Own", "own-v1")
	if _, err := s.settings.Update(func(v *settings.Settings) {
		if err := settings.ApplyKeyGame(v, "saveBackupHours", "6", "stardew"); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.profiles.SetOverride("stardew", own.ID, "saveBackupHours", "0"); err != nil {
		t.Fatal(err)
	}
	ownBackups := func() int {
		listed, err := s.ListBackups("stardew", "")
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, b := range listed {
			if b.Kind == backup.KindScheduled && b.Profile == own.ID {
				n++
			}
		}
		return n
	}
	now := time.Now()
	s.scheduledTickFor("stardew", now)
	if n := ownBackups(); n != 0 {
		t.Fatalf("a profile whose schedule is off was backed up %d times", n)
	}
	if _, err := s.profiles.SetOverride("stardew", own.ID, "saveBackupHours", "1"); err != nil {
		t.Fatal(err)
	}
	s.scheduledTickFor("stardew", now.Add(2*time.Hour))
	if n := ownBackups(); n == 0 {
		t.Fatal("a profile on an hourly schedule was not backed up two hours on, though the game's six hours had not passed")
	}
}
