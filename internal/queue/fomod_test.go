package queue

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// A downloaded archive with FOMOD options pauses the item, and the answer installs the chosen files through the real
// profile store; an answer to an item in any other state changes nothing.
func TestFomodArchivePausesAndAnswerInstalls(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "P")
	xml, err := os.ReadFile(filepath.Join("..", "installer", "testdata", "fomod", "choose-one.xml"))
	if err != nil {
		t.Fatal(err)
	}
	archive := testfs.ZipBytes(t, map[string]string{
		"fomod/ModuleConfig.xml": string(xml),
		"alpha/manifest.json":    `{"Name":"Alpha","Author":"a","Version":"1.0.0","UniqueID":"A.Alpha"}`,
		"beta/manifest.json":     `{"Name":"Beta","Author":"a","Version":"1.0.0","UniqueID":"A.Beta"}`,
	})

	f := newFixture(t)
	f.s.d.Install = func(ctx context.Context, game, id, path string, src profile.Source) (profile.InstallResult, error) {
		return profiles.InstallSource(ctx, game, id, path, src)
	}
	f.s.d.InstallStaged = profiles.InstallStaged
	f.cdn = func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archive) }
	f.start()
	f.add(Request{Kind: KindInstall, Game: "stardew", Profile: p.ID, ModID: 1, FileID: 10})

	st := f.wait("needs fomod", f.item(StateNeedsFomod))
	id := st.Items[0].ID
	if cur, err := profiles.Get("stardew", p.ID); err != nil || len(cur.Entries) != 0 {
		t.Fatalf("installed before the answer: %+v %v", cur.Entries, err)
	}

	f.s.AnswerFomod("nope", map[string]map[string][]string{"Options": {"Pack": {"Alpha"}}})
	if got := f.s.State().Items[0].State; got != StateNeedsFomod {
		t.Fatalf("answer to an unknown id moved the item to %s", got)
	}

	f.s.AnswerFomod(id, map[string]map[string][]string{"Options": {"Pack": {"Alpha"}}})
	f.wait("done", f.item(StateDone))
	cur, err := profiles.Get("stardew", p.ID)
	if err != nil || len(cur.Entries) != 1 || cur.Entries[0].Fomod["Options"]["Pack"][0] != "Alpha" {
		t.Fatalf("entries after the answer: %+v %v", cur.Entries, err)
	}
	if len(cur.Entries[0].Mods) != 1 || cur.Entries[0].Mods[0].Name != "Alpha" {
		t.Fatalf("installed mods: %+v", cur.Entries[0].Mods)
	}

	f.s.AnswerFomod(id, map[string]map[string][]string{"Options": {"Pack": {"Beta"}}})
	time.Sleep(20 * time.Millisecond)
	if got := f.s.State().Items[0]; got.State != StateDone || got.Fomod["Options"]["Pack"][0] != "Alpha" {
		t.Fatalf("answer to a done item changed it: %+v", got)
	}
}

func TestSkipAllLeavesDownloadsAndRetryFailedRequeues(t *testing.T) {
	t.Parallel()
	s := &Service{
		d:       Deps{Dir: t.TempDir(), Now: func() time.Time { return time.Unix(1, 0).UTC() }},
		kick:    make(chan struct{}, 1),
		cancels: map[string]context.CancelFunc{},
	}
	s.items = []*Item{
		{ID: "dl", State: StateDownloading},
		{ID: "q", State: StateQueued},
		{ID: "fomod", State: StateNeedsFomod, staged: "k"},
		{ID: "failed", State: StateFailed, Error: "boom", ErrorKind: FailOther},
		{ID: "done", State: StateDone},
	}
	s.SkipAll()
	want := map[string]string{"dl": StateDownloading, "q": StateSkipped, "fomod": StateSkipped, "failed": StateFailed, "done": StateDone}
	for id, state := range want {
		if got := s.find(id).State; got != state {
			t.Errorf("after SkipAll %s = %s, want %s", id, got, state)
		}
	}
	s.RetryFailed()
	want = map[string]string{"dl": StateDownloading, "q": StateQueued, "fomod": StateQueued, "failed": StateQueued, "done": StateDone}
	for id, state := range want {
		if got := s.find(id); got.State != state || (state == StateQueued && (got.Error != "" || got.ErrorKind != "")) {
			t.Errorf("after RetryFailed %+v, want %s", got, state)
		}
	}
}
