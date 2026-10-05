package sharesvc

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

func modZip(t *testing.T, uniqueID string) string {
	t.Helper()
	manifest := `{"Name":"Mod","Author":"a","Version":"1.0.0","UniqueID":"` + uniqueID + `","EntryDll":"Mod.dll"}`
	return testfs.WriteZip(t, filepath.Join(t.TempDir(), "mod.zip"), map[string]string{"Mod/manifest.json": manifest})
}

// pendingService is a signed-in service with its own state folder and an empty profile.
func pendingService(t *testing.T) (*Service, profile.Profile) {
	t.Helper()
	s, _ := newService(t, true)
	s.d.Dir = t.TempDir()
	return s, testenv.Profile(t, s.d.Profiles, "stardew", "P")
}

func TestPendingConfigsSurviveARestart(t *testing.T) {
	first, prof := pendingService(t)
	first.pending = []*pending{{
		Game: "stardew", Profile: prof.ID,
		Wanted:  []wantedFile{{ModID: 100, FileID: 1}},
		Configs: []share.Config{{UniqueID: "A.Mod", Path: "config.json", Data: []byte(`{"a":1}`)}},
	}}
	first.savePending()

	// Mortar quits before the mod finishes installing, and starts again.
	second := NewService(first.d)
	if len(second.pending) != 1 {
		t.Fatalf("pending after restart = %d, want 1", len(second.pending))
	}
	res, err := second.d.Profiles.InstallNexus("stardew", prof.ID, modZip(t, "A.Mod"), profile.Source{Kind: profile.KindNexus, ModID: 100, FileID: 1})
	if err != nil {
		t.Fatal(err)
	}
	done := queue.Item{ID: "a", Game: "stardew", Profile: prof.ID, ModID: 100, FileID: 1, State: queue.StateDone}
	second.QueueChanged(queue.State{Items: []queue.Item{done}})

	dir, err := second.d.Profiles.ModsDir("stardew", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	key := res.Profile.Entries[0].Key
	got, err := fsx.ReadFile(filepath.Join(dir, key, "Mod", "config.json"))
	if err != nil || string(got) != `{"a":1}` {
		t.Fatalf("config = %q, %v", got, err)
	}
	if len(second.pending) != 0 {
		t.Fatalf("settled import still pending: %d", len(second.pending))
	}
	if again := NewService(first.d); len(again.pending) != 0 {
		t.Fatalf("settled import came back after another restart")
	}
}

func TestPendingAppliesWhenTheDoneSetChangesAndSavesOnlyOnChange(t *testing.T) {
	s, prof := pendingService(t)
	s.pending = []*pending{{
		Game: "stardew", Profile: prof.ID,
		Wanted: []wantedFile{{ModID: 100, FileID: 1}, {ModID: 200, FileID: 2}, {ModID: 300, FileID: 3}},
		Configs: []share.Config{
			{UniqueID: "A.Mod", Path: "config.json", Data: []byte("a")},
			{UniqueID: "B.Mod", Path: "config.json", Data: []byte("b")},
		},
	}}
	s.savePending()
	file := filepath.Join(s.d.Dir, pendingFile)
	open := queue.Item{ID: "c", Game: "stardew", Profile: prof.ID, ModID: 300, FileID: 3, State: queue.StateDownloading}

	// A progress tick changes nothing and writes nothing.
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	s.QueueChanged(queue.State{Items: []queue.Item{open}})
	if _, err := os.Stat(file); err == nil {
		t.Fatal("pending imports saved on a change that touched none")
	}

	install := func(id string, modID, fileID int) string {
		res, err := s.d.Profiles.InstallNexus("stardew", prof.ID, modZip(t, id), profile.Source{Kind: profile.KindNexus, ModID: modID, FileID: fileID})
		if err != nil {
			t.Fatal(err)
		}
		return res.Profile.Entries[len(res.Profile.Entries)-1].Key
	}
	install("A.Mod", 100, 1)
	s.QueueChanged(queue.State{Items: []queue.Item{{ID: "a", Game: "stardew", Profile: prof.ID, ModID: 100, FileID: 1, State: queue.StateDone}, open}})
	// The queue dropped "a" as an old finished item while "b" finished: one done either way.
	bKey := install("B.Mod", 200, 2)
	s.QueueChanged(queue.State{Items: []queue.Item{{ID: "b", Game: "stardew", Profile: prof.ID, ModID: 200, FileID: 2, State: queue.StateDone}, open}})
	dir, err := s.d.Profiles.ModsDir("stardew", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := fsx.ReadFile(filepath.Join(dir, bKey, "Mod", "config.json")); err != nil || string(got) != "b" {
		t.Fatalf("config of the later mod = %q, %v", got, err)
	}
}

func TestPendingWaitsWhileTheGameRunsTheProfile(t *testing.T) {
	s, prof := pendingService(t)
	s.pending = []*pending{{
		Game: "stardew", Profile: prof.ID,
		Wanted:  []wantedFile{{ModID: 100, FileID: 1}},
		Configs: []share.Config{{UniqueID: "A.Mod", Path: "config.json", Data: []byte("a")}},
	}}
	res, err := s.d.Profiles.InstallNexus("stardew", prof.ID, modZip(t, "A.Mod"), profile.Source{Kind: profile.KindNexus, ModID: 100, FileID: 1})
	if err != nil {
		t.Fatal(err)
	}
	done := queue.State{Items: []queue.Item{{ID: "a", Game: "stardew", Profile: prof.ID, ModID: 100, FileID: 1, State: queue.StateDone}}}
	var running atomic.Bool
	running.Store(true)
	s.d.Profiles.Running = func(string, string) bool { return running.Load() }
	s.recheck = time.Hour
	s.QueueChanged(done)
	if len(s.pending) != 1 || s.pending[0].seen != "" || len(s.pending[0].Configs) != 1 {
		t.Fatalf("pending while running = %+v", s.pending)
	}

	running.Store(false)
	s.mu.Lock()
	if s.retry != nil {
		s.retry.Stop()
		s.retry = nil
	}
	s.mu.Unlock()
	s.retryPending()
	dir, err := s.d.Profiles.ModsDir("stardew", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := fsx.ReadFile(filepath.Join(dir, res.Profile.Entries[0].Key, "Mod", "config.json")); err != nil || string(got) != "a" {
		t.Fatalf("config after the game stopped = %q, %v", got, err)
	}
}

func TestPendingRetriesAfterInModsWriteError(t *testing.T) {
	s, prof := pendingService(t)
	s.pending = []*pending{{
		Game: "stardew", Profile: prof.ID,
		Wanted:  []wantedFile{{ModID: 100, FileID: 1}},
		Configs: []share.Config{{UniqueID: "A.Mod", Path: "config.json", Data: []byte("a")}},
	}}
	res, err := s.d.Profiles.InstallNexus("stardew", prof.ID, modZip(t, "A.Mod"), profile.Source{Kind: profile.KindNexus, ModID: 100, FileID: 1})
	if err != nil {
		t.Fatal(err)
	}
	dir, err := s.d.Profiles.ModsDir("stardew", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	modDir := filepath.Join(dir, res.Profile.Entries[0].Key, "Mod")
	block := filepath.Join(modDir, "config.json")
	if err := os.Mkdir(block, 0o700); err != nil {
		t.Fatal(err)
	}
	done := queue.State{Items: []queue.Item{{ID: "a", Game: "stardew", Profile: prof.ID, ModID: 100, FileID: 1, State: queue.StateDone}}}
	s.QueueChanged(done)
	if len(s.pending) != 1 || s.pending[0].seen != "" || len(s.pending[0].Configs) != 1 {
		t.Fatalf("pending after write error = %+v", s.pending)
	}
	if err := os.Remove(block); err != nil {
		t.Fatal(err)
	}
	s.QueueChanged(done)
	if got, err := fsx.ReadFile(block); err != nil || string(got) != "a" {
		t.Fatalf("config after retry = %q, %v", got, err)
	}
}

func TestPendingKeepsUnappliedConfigsAfterTheQueueDrains(t *testing.T) {
	s, prof := pendingService(t)
	s.pending = []*pending{{
		Game: "stardew", Profile: prof.ID,
		Wanted:  []wantedFile{{ModID: 100, FileID: 1}},
		Configs: []share.Config{{UniqueID: "Ghost.Mod", Path: "config.json", Data: []byte("g")}},
	}}
	if _, err := s.d.Profiles.InstallNexus("stardew", prof.ID, modZip(t, "A.Mod"), profile.Source{Kind: profile.KindNexus, ModID: 100, FileID: 1}); err != nil {
		t.Fatal(err)
	}
	s.QueueChanged(queue.State{Items: []queue.Item{
		{ID: "a", Game: "stardew", Profile: prof.ID, ModID: 100, FileID: 1, State: queue.StateDone},
	}})
	if len(s.pending) != 1 || len(s.pending[0].Configs) != 1 || s.pending[0].Configs[0].UniqueID != "Ghost.Mod" {
		t.Fatalf("pending dropped: %+v", s.pending)
	}
}

func TestPendingAppliesAfterFinishedQueueRowsAreGone(t *testing.T) {
	s, prof := pendingService(t)
	s.pending = []*pending{{
		Game: "stardew", Profile: prof.ID,
		Wanted:  []wantedFile{{ModID: 100, FileID: 1}},
		Configs: []share.Config{{UniqueID: "A.Mod", Path: "config.json", Data: []byte("a")}},
	}}
	res, err := s.d.Profiles.InstallNexus("stardew", prof.ID, modZip(t, "A.Mod"), profile.Source{Kind: profile.KindNexus, ModID: 100, FileID: 1})
	if err != nil {
		t.Fatal(err)
	}
	var running atomic.Bool
	running.Store(true)
	s.d.Profiles.Running = func(string, string) bool { return running.Load() }
	s.QueueChanged(queue.State{})
	if len(s.pending) != 1 || len(s.pending[0].Configs) != 1 {
		t.Fatalf("pending dropped while running with an empty queue: %+v", s.pending)
	}
	running.Store(false)
	s.QueueChanged(queue.State{})
	dir, err := s.d.Profiles.ModsDir("stardew", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := fsx.ReadFile(filepath.Join(dir, res.Profile.Entries[0].Key, "Mod", "config.json"))
	if err != nil || string(got) != "a" {
		t.Fatalf("config after empty-queue apply = %q, %v", got, err)
	}
}

func TestWantedFileItemMatchesGitHubAsset(t *testing.T) {
	w := wantedFile{Repo: "o/r", Tag: "1", Asset: "a.zip"}
	if !w.item(queue.Item{Repo: "o/r", Tag: "1", Asset: "a.zip"}) {
		t.Fatal("same asset should match")
	}
	if w.item(queue.Item{Repo: "o/r", Tag: "1", Asset: "b.zip"}) {
		t.Fatal("other asset from the same repo should not match")
	}
}
