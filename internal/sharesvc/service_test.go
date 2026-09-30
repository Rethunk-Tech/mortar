package sharesvc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/queue"
	"github.com/Rethunk-AI/mortar/internal/share"
	"github.com/Rethunk-AI/mortar/internal/store"
)

type recorder struct {
	reqs []queue.Request
	err  error
}

func (r *recorder) Add(reqs []queue.Request) ([]queue.Item, error) {
	r.reqs = append(r.reqs, reqs...)
	return nil, r.err
}

func newService(t *testing.T, signedIn bool) (*Service, *recorder) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	r := testResolver(map[int][]nexus.File{
		100: {nf(1, "1.0", "MAIN", true)},
		600: {nf(6, "1.0", "MAIN", true)},
		900: {nf(9, "1.2", "MAIN", true)},
	}, nil)
	rec := &recorder{}
	return NewService(Deps{
		Profiles: profiles, Meta: r.meta, Files: r.files, SignedIn: func() bool { return signedIn }, Premium: func() bool { return true },
		Env: func(string) problems.Environment { return problems.Environment{} }, Queue: rec,
	}), rec
}

func link(t *testing.T, name string, refs ...share.Ref) string {
	t.Helper()
	p := profile.Profile{Name: name}
	for i, r := range refs {
		e := profile.Entry{Key: "k" + string(rune('a'+i))}
		if r.GitHub != "" {
			repo, tag, asset := r.GitHubParts()
			e.Source = profile.Source{Kind: profile.KindGitHub, Repo: repo, Tag: tag, Asset: asset}
		} else {
			e.Source = profile.Source{Kind: profile.KindNexus, ModID: r.ModID, FileID: r.FileID}
		}
		p.Entries = append(p.Entries, e)
	}
	res, err := share.Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	return res.App
}

func TestImportCreatesNothingBeforeConfirmAndQueuesAvailable(t *testing.T) {
	s, rec := newService(t, true)
	text := link(t, "Cozy co-op", share.Ref{ModID: 100, FileID: 1}, share.Ref{ModID: 500, FileID: 5}, share.Ref{ModID: 700, FileID: 7}, share.Ref{GitHub: "o/r@v1/a.zip"})
	pv, err := s.PreviewLink(context.Background(), "stardew", text, "")
	if err != nil || pv.Name != "Cozy co-op" {
		t.Fatalf("preview = %+v, %v", pv, err)
	}
	if all, _ := s.d.Profiles.List("stardew"); len(all) != 0 || len(rec.reqs) != 0 {
		t.Fatalf("preview changed something: %d profiles, %d requests", len(all), len(rec.reqs))
	}
	skip := byKey(pv.Mods, Mod{ModID: 700}).Key
	res, err := s.Import("stardew", "", []string{skip})
	if err != nil {
		t.Fatal(err)
	}
	// 100 and the GitHub asset are wanted, 700 is excluded, 500 is unavailable, and 900 comes in as 100's dependency.
	var got []string
	for _, r := range rec.reqs {
		if r.Profile != res.Profile.ID || r.Game != "stardew" {
			t.Errorf("request for the wrong profile: %+v", r)
		}
		got = append(got, fmt.Sprintf("%s:%s:%d", r.Kind, r.Repo, r.ModID/100))
	}
	slices.Sort(got)
	want := []string{"dependency::9", "install:o/r:0", "install::1"}
	slices.Sort(want)
	if !slices.Equal(got, want) || res.Queued != 3 {
		t.Errorf("requests = %v, queued %d", got, res.Queued)
	}
	if !strings.Contains(res.Profile.Notes, "Nexus mod 500") || !strings.Contains(res.Profile.Notes, "/mods/500") {
		t.Errorf("notes = %q", res.Profile.Notes)
	}
	if _, err := s.Import("stardew", "", nil); !errors.Is(err, ErrNoPreview) {
		t.Errorf("second import = %v", err)
	}
}

func TestImportNeedsSignInForNexusAndCreatesNothing(t *testing.T) {
	s, rec := newService(t, false)
	text := link(t, "Cozy", share.Ref{ModID: 100, FileID: 1})
	if _, err := s.PreviewLink(context.Background(), "stardew", text, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import("stardew", "", nil); !errors.Is(err, ErrSignedOut) {
		t.Fatalf("err = %v", err)
	}
	if all, _ := s.d.Profiles.List("stardew"); len(all) != 0 || len(rec.reqs) != 0 {
		t.Errorf("a refused import left %d profiles, %d requests", len(all), len(rec.reqs))
	}
}

func TestImportGitHubOnlyNeedsNoSignIn(t *testing.T) {
	s, rec := newService(t, false)
	text := link(t, "Gh", share.Ref{GitHub: "o/r@v1/a.zip"})
	if _, err := s.PreviewLink(context.Background(), "stardew", text, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import("stardew", "", nil); err != nil || len(rec.reqs) != 1 || rec.reqs[0].Tag != "v1" {
		t.Errorf("reqs = %+v, %v", rec.reqs, err)
	}
}

func TestImportRollsBackANewProfileWhenQueueingFails(t *testing.T) {
	s, rec := newService(t, true)
	rec.err = errors.New("nope")
	text := link(t, "Cozy", share.Ref{ModID: 100, FileID: 1})
	if _, err := s.PreviewLink(context.Background(), "stardew", text, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import("stardew", "", nil); err == nil {
		t.Fatal("import succeeded")
	}
	if all, _ := s.d.Profiles.List("stardew"); len(all) != 0 {
		t.Errorf("%d profiles left", len(all))
	}
}

func TestDiscardLeavesNothingToImport(t *testing.T) {
	s, _ := newService(t, true)
	text := link(t, "Cozy", share.Ref{ModID: 100, FileID: 1})
	if _, err := s.PreviewLink(context.Background(), "stardew", text, ""); err != nil {
		t.Fatal(err)
	}
	s.Discard()
	if _, err := s.Import("stardew", "", nil); !errors.Is(err, ErrNoPreview) {
		t.Errorf("err = %v", err)
	}
}

func TestParseErrorsSurface(t *testing.T) {
	s, _ := newService(t, true)
	if _, err := s.PreviewLink(context.Background(), "stardew", "hello", ""); !errors.Is(err, share.ErrNotLink) {
		t.Errorf("err = %v", err)
	}
}

func TestReceiveRoutesLinksAndFiles(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Cozy farm.mortar")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	var events []Arrival
	s := NewService(Deps{Emit: func(name string, data any) {
		if a, ok := data.(Arrival); ok && name == ArrivedEvent {
			events = append(events, a)
		}
	}})
	args := []string{
		"/usr/bin/mortar", "mortar://stardew/p/abc", "https://mortar.rethunk.tech/stardew/p#abc", "nxm://stardewvalley/mods/1/files/2",
		file, "file://" + file, filepath.Join(dir, "missing.mortar"), "notes.txt", "https://evil.example/stardew/p#abc",
	}
	if !s.Receive(args) {
		t.Fatal("Receive found nothing")
	}
	want := []Arrival{
		{ID: 1, Kind: ArrivalLink, Value: "mortar://stardew/p/abc"},
		{ID: 2, Kind: ArrivalLink, Value: "https://mortar.rethunk.tech/stardew/p#abc"},
		{ID: 3, Kind: ArrivalFile, Value: file},
		{ID: 4, Kind: ArrivalFile, Value: file},
	}
	if !slices.Equal(events, want) {
		t.Errorf("events = %+v", events)
	}
	if got := s.Inbox(); !slices.Equal(got, want) {
		t.Errorf("inbox = %+v", got)
	}
	if got := s.Inbox(); len(got) != 0 {
		t.Errorf("inbox is handed over once, got %+v", got)
	}
	if s.Receive([]string{"/usr/bin/mortar"}) {
		t.Error("plain launch counted as a link")
	}
}

func TestDescribeGroupsAndLeftOut(t *testing.T) {
	nx := func(key string, mod int, name string, disabled ...string) profile.Entry {
		return profile.Entry{
			Key: key, Source: profile.Source{Kind: profile.KindNexus, ModID: mod, FileID: 1},
			Mods: []profile.EntryMod{{UniqueID: "U." + key, Name: name}}, Disabled: disabled,
		}
	}
	p := profile.Profile{Name: "Farm", Entries: []profile.Entry{
		{Key: "smapi-4", Source: profile.Source{Kind: profile.SourceSMAPI}},
		nx("a", 1, "Alpha"), nx("b", 2, "Beta", "U.b"),
		{Key: "loc", Source: profile.Source{Kind: profile.KindLocal, Name: "mine.zip"}},
		{Key: "gh", Source: profile.Source{Kind: profile.KindGitHub, Repo: "o/r", Tag: "v1", Asset: "a.zip"}, Mods: []profile.EntryMod{{UniqueID: "G", Name: "Gee"}}},
	}}
	info, err := describe(p)
	if err != nil || info.TooLarge || info.Count != 2 || info.Length != len(info.Web) || info.Limit != DiscordLimit {
		t.Fatalf("info = %+v, %v", info, err)
	}
	wantGroups := []Group{{Source: "nexus", Mods: []string{"Alpha"}}, {Source: "github", Mods: []string{"Gee"}}}
	if !slices.EqualFunc(info.Groups, wantGroups, func(a, b Group) bool { return a.Source == b.Source && slices.Equal(a.Mods, b.Mods) }) {
		t.Errorf("groups = %+v", info.Groups)
	}
	wantLeft := []Omitted{{Name: "mine.zip", Reason: "local"}, {Name: "Beta", Reason: "off"}}
	if !slices.Equal(info.LeftOut, wantLeft) {
		t.Errorf("left out = %+v", info.LeftOut)
	}
}
