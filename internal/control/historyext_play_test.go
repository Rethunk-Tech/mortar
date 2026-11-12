package control

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// historyHarness gives the services a profile holding count Thunderstore packages, so its history has one change per
// package, and a launch service over the same stores.
func historyHarness(t *testing.T, count int) (*Services, profile.Profile) {
	t.Helper()
	s, items := servicesWithItems(t)
	const gameID = "lethal-company"
	prof, err := s.Profiles.Create(gameID, "Crew")
	if err != nil {
		t.Fatal(err)
	}
	for i := range count {
		files := t.TempDir()
		name := fmt.Sprintf("Package%d", i)
		manifest := fmt.Sprintf(`{"name":%q,"namespace":"Ns","version_number":"1.0.0","dependencies":[]}`, name)
		for file, body := range map[string]string{name + ".dll": "x", "manifest.json": manifest} {
			if err := os.WriteFile(filepath.Join(files, file), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		key := store.PackageKey("Ns-"+name, "1.0.0")
		if err := items.AddDir(t.Context(), gameID, key, files); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Store.AddEntry(gameID, prof.ID, key, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-" + name, Version: "1.0.0"}); err != nil {
			t.Fatal(err)
		}
	}
	s.Launches = launchsvc.NewService(t.TempDir(), s.Settings, s.Store)
	return s, prof
}

// recordRun writes the profile's run index with one run that started at started.
func recordRun(t *testing.T, s *Services, prof profile.Profile, started time.Time) {
	t.Helper()
	modsDir, err := s.Profiles.ModsDir("lethal-company", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(filepath.Dir(modsDir), "runs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	index := struct {
		Runs []struct {
			ID      string `json:"id"`
			Started string `json:"started"`
		} `json:"runs"`
	}{}
	index.Runs = append(index.Runs, struct {
		ID      string `json:"id"`
		Started string `json:"started"`
	}{"r1", started.Format(time.RFC3339Nano)})
	if err := datadir.WriteJSON(filepath.Join(dir, "index.json"), index); err != nil {
		t.Fatal(err)
	}
}

func TestProfileChangesStopsAtTheNewestRun(t *testing.T) {
	s, prof := historyHarness(t, 1)
	recordRun(t, s, prof, time.Now().Add(-time.Hour))
	all, err := s.profileChanges("lethal-company", prof.ID)
	if err != nil || len(all.Items) == 0 {
		t.Fatalf("a run older than every change: %+v %v", all, err)
	}
	recordRun(t, s, prof, time.Now().Add(time.Hour))
	since, err := s.profileChanges("lethal-company", prof.ID)
	if err != nil || len(since.Items) != 0 {
		t.Fatalf("a run newer than every change: %+v %v", since, err)
	}
}

func TestProfileGoodRefusesBothFlagsWithTheCLIWords(t *testing.T) {
	s, prof := historyHarness(t, 1)
	_, err := s.profileGood("lethal-company", prof.ID, true, true)
	if err == nil || err.Error() != "profile good takes --mark or --restore, not both" {
		t.Fatalf("err = %v", err)
	}
}
