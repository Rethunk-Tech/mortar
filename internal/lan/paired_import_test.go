package lan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// noMeta is a mod dataset that knows nothing, so previews rest on the Nexus file list alone.
type noMeta struct{}

func (noMeta) Lookup(context.Context, string) ([]meta.Ref, error) { return nil, nil }
func (noMeta) Page(context.Context, int) (meta.Page, error) {
	return meta.Page{}, errors.New("no page")
}

func (noMeta) Collection(context.Context, string, string, int) (meta.Collection, error) {
	return meta.Collection{}, errors.New("no collection")
}
func (noMeta) CheckUpdates(context.Context, meta.UpdateRequest) []meta.UpdateResult { return nil }

type queueSpy struct{ reqs atomic.Int64 }

func (q *queueSpy) Add(_ context.Context, reqs []queue.Request) ([]queue.Item, error) {
	q.reqs.Add(int64(len(reqs)))
	return nil, nil
}

func smapiFolder(t *testing.T, id string) string {
	t.Helper()
	dir := t.TempDir()
	manifest := fmt.Sprintf(`{"Name":%q,"Author":"a","Version":"1.0.0","UniqueID":%q,"EntryDll":"m.dll"}`, id, id)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// A paired receiver gets every source's files from the sender and places them: nothing is queued, so nothing is
// downloaded, and an unpaired receiver of the same profile queues the Nexus and GitHub files.
func TestPairedSendPlacesEverySourceWithoutDownloading(t *testing.T) {
	const game = "stardew"
	senderStore := store.OpenAt(t.TempDir())
	nexusKey, ghKey, localKey := store.NexusKey(7, 2), github.Key("o", "r", "v1", "m.zip"), store.LocalKey(strings.Repeat("cd", 32))
	for key, id := range map[string]string{nexusKey: "Me.Nexus", ghKey: "Me.GitHub", localKey: "Me.Local"} {
		if err := senderStore.AddDir(t.Context(), game, key, smapiFolder(t, id)); err != nil {
			t.Fatal(err)
		}
	}
	var payload bytes.Buffer
	if _, err := share.Write(&payload, game, profile.Profile{Name: "Farm friends", Entries: []profile.Entry{
		{Key: nexusKey, Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 2}},
		{Key: ghKey, Source: profile.Source{Kind: profile.KindGitHub, Repo: "o/r", Tag: "v1", Asset: "m.zip"}},
		{Key: localKey, Source: profile.Source{Kind: profile.KindLocal, Name: "Mine.zip"}},
	}}, t.TempDir(), share.Include{LocalFiles: true}); err != nil {
		t.Fatal(err)
	}

	testfs.DataHome(t)
	items, profiles := testenv.Stores(t)
	spy := &queueSpy{}
	var nexusCalls atomic.Int64
	shares := sharesvc.NewService(sharesvc.Deps{
		Profiles: profiles, Meta: noMeta{}, Queue: spy, SignedIn: func() bool { return true }, Premium: func() bool { return false },
		Files: func(context.Context, nexus.Title, int) ([]nexus.File, error) {
			nexusCalls.Add(1)
			return []nexus.File{{FileID: 2, FileName: "n.zip", Version: "1.0.0", Category: "MAIN", IsPrimary: true}}, nil
		},
		Env: func(string) problems.Environment {
			return problems.Environment{Nexus: nexus.Title{Domain: "stardewvalley", ID: 1303}, VersionScheme: deps.SemverSMAPI}
		},
		Stored: func(g, k string) bool { _, err := items.Path(g, k); return err == nil },
	})

	arrivals := make(chan Arrival, 1)
	receiver, receiverAddr := pairedService(t, items, func(_ string, data any) {
		if a, ok := data.(Arrival); ok {
			arrivals <- a
		}
	})
	sender, _ := pairedService(t, senderStore, nil)
	build := fixed(payload.Bytes())
	next := func() Arrival {
		t.Helper()
		if err := sender.sendPayload(t.Context(), receiverAddr, game, "", build); err != nil {
			t.Fatal(err)
		}
		select {
		case a := <-arrivals:
			return a
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for share")
			return Arrival{}
		}
	}

	unpaired := next()
	if _, err := shares.ImportData(t.Context(), game, unpaired.Payload); err != nil {
		t.Fatal(err)
	}
	if spy.reqs.Load() != 2 {
		t.Fatalf("unpaired import queued %d requests, want the Nexus and GitHub files", spy.reqs.Load())
	}
	spy.reqs.Store(0)

	pair(t, receiver, sender, receiverAddr)
	receiver.rateMu.Lock()
	receiver.lastReceive = map[string]time.Time{}
	receiver.rateMu.Unlock()
	arrival := next()
	if !arrival.Paired {
		t.Fatal("paired share got no transfer grant")
	}
	if err := receiver.Transfer(t.Context(), arrival.ID); err != nil {
		t.Fatal(err)
	}
	res, err := shares.ImportData(t.Context(), game, arrival.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if n := spy.reqs.Load(); n != 0 {
		t.Fatalf("paired import queued %d downloads", n)
	}
	var kinds []string
	for _, e := range res.Profile.Entries {
		kinds = append(kinds, e.Source.Kind)
	}
	slices.Sort(kinds)
	if want := []string{profile.KindGitHub, profile.KindLocal, profile.KindNexus}; !slices.Equal(kinds, want) {
		t.Fatalf("placed entries = %v, want %v", kinds, want)
	}
}
