package packsvc

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
)

type fakeProfiles struct{ created []string }

func (f *fakeProfiles) Create(_, name string) (profile.Profile, error) {
	f.created = append(f.created, name)
	return profile.Profile{ID: "p1", Name: name}, nil
}

func (*fakeProfiles) List(string) ([]profile.Profile, error) { return nil, nil }

type fakeQueue struct{ got []queue.Request }

func (f *fakeQueue) Add(reqs []queue.Request) ([]queue.Item, error) {
	f.got = append(f.got, reqs...)
	return make([]queue.Item, len(reqs)), nil
}

func writeR2z(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "friends.r2z")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("export.r2x")
	_, _ = w.Write([]byte(`profileName: Friends
mods:
  - name: BepInEx-BepInExPack
    version: {major: 5, minor: 4, patch: 2100}
    enabled: true
  - name: Alice-MoreCompany
    version: {major: 1, minor: 2, patch: 3}
    enabled: false
`))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImportQueuesThePacksPackagesIntoANewProfile(t *testing.T) {
	ps, q := &fakeProfiles{}, &fakeQueue{}
	s := &Service{Profiles: ps, Queue: q}
	src := Source{Path: writeR2z(t)}
	pv, err := s.Preview(context.Background(), src)
	if err != nil || pv.Name != "Friends" || len(pv.Packages) != 2 {
		t.Fatalf("preview %+v, %v", pv, err)
	}
	if _, err := s.Import(context.Background(), src, "", ""); err == nil {
		t.Error("a pack without a game imported")
	}
	res, err := s.Import(context.Background(), src, "stardew", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Queued != 2 || res.Profile != "p1" || !slices.Equal(res.Disabled, []string{"Alice-MoreCompany"}) || !slices.Equal(ps.created, []string{"Friends"}) {
		t.Errorf("result %+v, created %v", res, ps.created)
	}
	if len(q.got) != 2 || q.got[1].Package != "Alice-MoreCompany" || q.got[1].Version != "1.2.3" || q.got[1].Profile != "p1" || q.got[1].Game != "stardew" {
		t.Errorf("queued %+v", q.got)
	}
}

func TestExportNeedsTheConfirmation(t *testing.T) {
	if _, err := (&Service{}).ExportCode(context.Background(), "stardew", "p1", false); err == nil {
		t.Error("an unconfirmed export ran")
	}
}
