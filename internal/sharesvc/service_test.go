package sharesvc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

type recorder struct {
	reqs []queue.Request
	err  error
}

func (r *recorder) Add(_ context.Context, reqs []queue.Request) ([]queue.Item, error) {
	r.reqs = append(r.reqs, reqs...)
	return nil, r.err
}

func newService(t *testing.T, signedIn bool) (*Service, *recorder) {
	t.Helper()
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	r := testResolver(map[int][]nexus.File{
		100: {nf(1, "1.0", "MAIN", true)},
		600: {nf(6, "1.0", "MAIN", true)},
		900: {nf(9, "1.2", "MAIN", true)},
	}, nil)
	rec := &recorder{}
	return NewService(Deps{
		Profiles: profiles, Meta: r.meta, Files: r.files, SignedIn: func() bool { return signedIn }, Premium: func() bool { return true },
		Env: func(string) problems.Environment { return stardewEnv }, Queue: rec,
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
	res, err := share.Encode("stardew", p, profile.ShareFacts{})
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
	res, err := s.Import(context.Background(), "stardew", sessionOf(s), "", []string{skip})
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
	if _, err := s.Import(context.Background(), "stardew", sessionOf(s), "", nil); !errors.Is(err, ErrNoPreview) {
		t.Errorf("second import = %v", err)
	}
}

func TestImportNamesAClashingProfileWithANumber(t *testing.T) {
	s, _ := newService(t, true)
	if _, err := s.d.Profiles.Create("stardew", "Cozy co-op"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PreviewLink(context.Background(), "stardew", link(t, "Cozy co-op", share.Ref{ModID: 100, FileID: 1}), ""); err != nil {
		t.Fatal(err)
	}
	res, err := s.Import(context.Background(), "stardew", sessionOf(s), "", nil)
	if err != nil || res.Profile.Name != "Cozy co-op (2)" {
		t.Fatalf("import = %q, %v", res.Profile.Name, err)
	}
}

func TestImportNeedsSignInForNexusAndCreatesNothing(t *testing.T) {
	s, rec := newService(t, false)
	text := link(t, "Cozy", share.Ref{ModID: 100, FileID: 1})
	if _, err := s.PreviewLink(context.Background(), "stardew", text, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import(context.Background(), "stardew", sessionOf(s), "", nil); !errors.Is(err, ErrSignedOut) {
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
	if _, err := s.Import(context.Background(), "stardew", sessionOf(s), "", nil); err != nil || len(rec.reqs) != 1 || rec.reqs[0].Tag != "v1" {
		t.Errorf("reqs = %+v, %v", rec.reqs, err)
	}
	all, err := s.d.Profiles.List("stardew")
	if err != nil || len(all) != 1 || all[0].Origin != profile.OriginLink {
		t.Errorf("origin = %+v, %v", all, err)
	}
}

func TestImportFromMortarFileRecordsOrigin(t *testing.T) {
	s, rec := newService(t, false)
	src := profile.Profile{
		Name: "From file",
		Entries: []profile.Entry{
			{
				Key: "gh", Source: profile.Source{Kind: profile.KindGitHub, Repo: "o/r", Tag: "v1", Asset: "a.zip"},
				Mods: []profile.Component{{ID: "smapi:G.Mod", Name: "Gee", Folder: "."}},
			},
		},
	}
	file := filepath.Join(t.TempDir(), "from.mortar")
	f, err := fsx.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := share.Write(f, "stardew", src, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PreviewFile(context.Background(), "stardew", file, ""); err != nil {
		t.Fatal(err)
	}
	res, err := s.Import(context.Background(), "stardew", sessionOf(s), "", nil)
	if err != nil || res.Profile.Origin != profile.OriginMortar || len(rec.reqs) != 1 {
		t.Fatalf("import = %+v queued %d: %v", res.Profile, len(rec.reqs), err)
	}
}

func TestImportRollsBackANewProfileWhenQueueingFails(t *testing.T) {
	s, rec := newService(t, true)
	rec.err = errors.New("nope")
	text := link(t, "Cozy", share.Ref{ModID: 100, FileID: 1})
	if _, err := s.PreviewLink(context.Background(), "stardew", text, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import(context.Background(), "stardew", sessionOf(s), "", nil); err == nil {
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
	if _, err := s.Import(context.Background(), "stardew", sessionOf(s), "", nil); !errors.Is(err, ErrNoPreview) {
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
	nx := func(key string, im int, name string, disabled ...mod.ID) profile.Entry {
		return profile.Entry{
			Key: key, Source: profile.Source{Kind: profile.KindNexus, ModID: im, FileID: 1},
			Mods: []profile.Component{{ID: mod.SMAPI("U." + key), Name: name}}, Disabled: disabled,
		}
	}
	p := profile.Profile{Name: "Farm", Entries: []profile.Entry{
		{Key: "smapi-4", Source: profile.Source{Kind: profile.SourceSMAPI}},
		nx("a", 1, "Alpha"), nx("b", 2, "Beta", "smapi:U.b"),
		{Key: "loc", Source: profile.Source{Kind: profile.KindLocal, Name: "mine.zip"}},
		{Key: "gh", Source: profile.Source{Kind: profile.KindGitHub, Repo: "o/r", Tag: "v1", Asset: "a.zip"}, Mods: []profile.Component{{ID: "smapi:G", Name: "Gee"}}},
	}}
	info, err := describe("stardew", p, profile.ShareFacts{})
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
	subset, err := describe("stardew", withEntryKeys(p, []string{"a"}), profile.ShareFacts{})
	if err != nil || subset.Count != 1 {
		t.Fatalf("subset = %+v, %v", subset, err)
	}
	if len(subset.Groups) != 1 || !slices.Equal(subset.Groups[0].Mods, []string{"Alpha"}) {
		t.Errorf("subset groups = %+v", subset.Groups)
	}
}

func TestDescribeCountsThunderstorePackages(t *testing.T) {
	p := profile.Profile{Name: "Lobby", Entries: []profile.Entry{
		{Key: "ts", Source: profile.Source{Kind: profile.KindThunderstore, Name: "x753-More_Suits", Version: "1.5.4"}, Mods: []profile.Component{{ID: "thunderstore:x753-More_Suits", Name: "More_Suits"}}},
	}}
	info, err := describe("lethal-company", p, profile.ShareFacts{})
	if err != nil || info.Count != 1 || len(info.Groups) != 1 || info.Groups[0].Source != profile.KindThunderstore {
		t.Fatalf("a Thunderstore package is counted and grouped: %+v, %v", info, err)
	}
	got, err := share.Parse(info.App)
	if err != nil || len(got.Entries) != 1 || got.Entries[0].Package != "x753-More_Suits" || got.Entries[0].Version != "1.5.4" {
		t.Fatalf("the link carries the package: %+v, %v", got, err)
	}
}

func sessionOf(s *Service) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return ""
	}
	return s.current.id
}

func TestImportActsOnlyOnTheShownPreviewAndOnlyOnce(t *testing.T) {
	s, rec := newService(t, true)
	first, err := s.PreviewLink(context.Background(), "stardew", link(t, "First", share.Ref{ModID: 100, FileID: 1}), "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.PreviewLink(context.Background(), "stardew", link(t, "Second", share.Ref{ModID: 600, FileID: 6}), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import(context.Background(), "stardew", first.Session, "", nil); !errors.Is(err, ErrStalePreview) {
		t.Fatalf("import of a replaced preview = %v", err)
	}
	res, err := s.Import(context.Background(), "stardew", second.Session, "", nil)
	if err != nil || res.Profile.Name != "Second" {
		t.Fatalf("import = %+v, %v", res, err)
	}
	if _, err := s.Import(context.Background(), "stardew", second.Session, "", nil); !errors.Is(err, ErrNoPreview) {
		t.Fatalf("second submit = %v", err)
	}
	if all, _ := s.d.Profiles.List("stardew"); len(all) != 1 || len(rec.reqs) != 1 {
		t.Fatalf("%d profiles, %d requests", len(all), len(rec.reqs))
	}
}

func TestImportLeavesTheConfigOfModsTheProfileHas(t *testing.T) {
	s, _ := newService(t, true)
	prof, err := s.d.Profiles.Create("stardew", "Mine")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.d.Profiles.InstallSource(t.Context(), "stardew", prof.ID, modZip(t, "A.Mod"), profile.Source{Kind: profile.KindNexus, ModID: 100, FileID: 1}); err != nil {
		t.Fatal(err)
	}
	s.current = &session{
		id: "s", game: "stardew", target: prof.ID,
		preview: Preview{Mods: []Mod{{Key: "a", ModID: 100, FileID: 2}, {Key: "b", ModID: 600, FileID: 6}}},
		configs: []share.Config{{ID: "smapi:a.mod", Path: "config.json"}, {ID: "smapi:B.Mod", Path: "config.json"}},
	}
	if _, err := s.Import(context.Background(), "stardew", "s", prof.ID, nil); err != nil {
		t.Fatal(err)
	}
	if len(s.pending) != 1 || len(s.pending[0].Configs) != 1 || s.pending[0].Configs[0].ID != "smapi:B.Mod" {
		t.Fatalf("pending = %+v", s.pending)
	}
}

// slowFirst makes the first Files call wait until release is closed, reporting on entered once it is waiting.
func slowFirst(s *Service) (entered, release chan struct{}) {
	entered, release = make(chan struct{}), make(chan struct{})
	files := s.d.Files
	var first atomic.Bool
	s.d.Files = func(ctx context.Context, t nexus.Title, modID int) ([]nexus.File, error) {
		if first.CompareAndSwap(false, true) {
			close(entered)
			<-release
		}
		return files(ctx, t, modID)
	}
	return entered, release
}

func TestAnOvertakenPreviewDoesNotReplaceTheNewer(t *testing.T) {
	for _, discard := range []bool{false, true} {
		s, _ := newService(t, true)
		entered, release := slowFirst(s)
		done := make(chan error)
		go func() {
			_, err := s.PreviewLink(context.Background(), "stardew", link(t, "Old", share.Ref{ModID: 100, FileID: 1}), "")
			done <- err
		}()
		<-entered
		want := ""
		if discard {
			s.Discard()
		} else {
			pv, err := s.PreviewLink(context.Background(), "stardew", link(t, "New", share.Ref{ModID: 100, FileID: 1}), "")
			if err != nil {
				t.Fatal(err)
			}
			want = pv.Session
		}
		close(release)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if got := sessionOf(s); got != want {
			t.Errorf("discard %v: current = %q, want %q", discard, got, want)
		}
	}
}

func TestANewProfileFromAPreviewOfTheOpenOneGetsItsMods(t *testing.T) {
	s, rec := newService(t, true)
	prof, err := s.d.Profiles.Create("stardew", "Mine")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.d.Profiles.InstallSource(t.Context(), "stardew", prof.ID, modZip(t, "A.Mod"), profile.Source{Kind: profile.KindNexus, ModID: 100, FileID: 1}); err != nil {
		t.Fatal(err)
	}
	pv, err := s.PreviewLink(context.Background(), "stardew", link(t, "Cozy", share.Ref{ModID: 100, FileID: 1}), prof.ID)
	if err != nil || len(pv.Mods) != 1 || pv.Mods[0].State != StateInstalled {
		t.Fatalf("preview against the open profile = %+v, %v", pv.Mods, err)
	}
	if _, err := s.Import(context.Background(), "stardew", pv.Session, "", nil); err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(rec.reqs, func(r queue.Request) bool { return r.ModID == 100 }) {
		t.Fatalf("queued %+v", rec.reqs)
	}
}

func TestInDirResolvesOnlyRelativePaths(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "a.mortar")
	got := InDir([]string{"farm.mortar", abs, "mortar://stardew/p/abc", "file://" + abs}, filepath.FromSlash("/home/u"))
	want := []string{filepath.Join(filepath.FromSlash("/home/u"), "farm.mortar"), abs, "mortar://stardew/p/abc", "file://" + abs}
	if !slices.Equal(got, want) {
		t.Errorf("InDir = %q", got)
	}
}

func TestRequestForCarriesOverlay(t *testing.T) {
	r := requestFor("stardew", "p", Mod{Site: SiteNexus, ModID: 7, FileID: 2, Overlay: &share.Overlay{From: "a", To: "b", Off: true}})
	if r.Overlay == nil || *r.Overlay != (queue.OverlayPlace{From: "a", To: "b", Off: true}) {
		t.Fatalf("overlay = %+v", r.Overlay)
	}
}
