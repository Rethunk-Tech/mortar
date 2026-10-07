package store

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestReport(t *testing.T) {
	const gameID = "stardew"
	cases := []struct {
		name       string
		items      []testItem
		referenced []string
		unusedKeys []string
		dupGroups  int
		dupSize    int
	}{
		{
			name: "unused",
			items: []testItem{
				{key: "local-aaa", uniqueID: "A.Mod", version: "1.0.0", name: "Alpha"},
			},
			unusedKeys: []string{"local-aaa"},
		},
		{
			name: "referenced",
			items: []testItem{
				{key: "local-bbb", uniqueID: "B.Mod", version: "1.0.0", name: "Beta"},
			},
			referenced: []string{"local-bbb"},
		},
		{
			name: "duplicates",
			items: []testItem{
				{key: "local-ccc", uniqueID: "C.Mod", version: "2.0.0", name: "Copy"},
				{key: "nexus-1-2", uniqueID: "C.Mod", version: "2.0.0", name: "Copy"},
				{key: "local-ddd", uniqueID: "D.Mod", version: "1.0.0", name: "Other"},
			},
			unusedKeys: []string{"local-ccc", "local-ddd", "nexus-1-2"},
			dupGroups:  1,
			dupSize:    2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Store{root: t.TempDir()}
			for _, it := range tc.items {
				addTestItem(t, s, gameID, it)
			}
			rep, err := s.Report(map[string][]string{gameID: tc.referenced})
			if err != nil {
				t.Fatal(err)
			}
			got := rep[gameID]
			if keysOf(got.Unused) != joinKeys(tc.unusedKeys) {
				t.Fatalf("unused keys: got %s want %s", keysOf(got.Unused), joinKeys(tc.unusedKeys))
			}
			if len(got.Duplicates) != tc.dupGroups {
				t.Fatalf("duplicate groups: got %d want %d", len(got.Duplicates), tc.dupGroups)
			}
			if tc.dupSize > 0 && (len(got.Duplicates) == 0 || len(got.Duplicates[0]) != tc.dupSize) {
				t.Fatalf("duplicate group size: got %v want %d", got.Duplicates, tc.dupSize)
			}
			for _, u := range got.Unused {
				if u.LastUsed.IsZero() {
					t.Fatalf("unused %s missing LastUsed", u.Key)
				}
				if u.Size <= 0 {
					t.Fatalf("unused %s size %d", u.Key, u.Size)
				}
			}
		})
	}
}

type testItem struct {
	key, uniqueID, version, name string
}

func addTestItem(t *testing.T, s *Store, gameID string, it testItem) {
	t.Helper()
	src := t.TempDir()
	body := `{"Name":"` + it.name + `","UniqueID":"` + it.uniqueID + `","Version":"` + it.version + `"}`
	testfs.WriteFile(t, src, "manifest.json", body)
	if err := s.AddDir(t.Context(), gameID, it.key, src); err != nil {
		t.Fatal(err)
	}
}

func keysOf(items []Item) string {
	keys := make([]string, len(items))
	for i, it := range items {
		keys[i] = it.Key
	}
	return joinKeys(keys)
}

func joinKeys(keys []string) string {
	out := ""
	for i, k := range keys {
		if i > 0 {
			out += ","
		}
		out += k
	}
	return out
}

func TestReportMarksInUseDuplicates(t *testing.T) {
	s := &Store{root: t.TempDir()}
	addTestItem(t, s, "stardew", testItem{key: "nexus-1-1", uniqueID: "C.Mod", version: "2.0.0", name: "Copy"})
	addTestItem(t, s, "stardew", testItem{key: "nexus-1-2", uniqueID: "C.Mod", version: "2.0.0", name: "Copy"})
	rep, err := s.Report(map[string][]string{"stardew": {"nexus-1-1"}})
	if err != nil {
		t.Fatal(err)
	}
	g := rep["stardew"].Duplicates[0]
	if !g[0].InUse || g[1].InUse {
		t.Fatalf("inUse flags: %+v", g)
	}
}

func TestReportLeavesALoaderItemUnnamedAndOutOfDuplicates(t *testing.T) {
	s := &Store{root: t.TempDir()}
	addTestItem(t, s, "lethal-company", testItem{key: "bepinex5-5.4.2304", uniqueID: "A.Mod", version: "1.0.0", name: "Alpha"})
	addTestItem(t, s, "lethal-company", testItem{key: "bepinex5-5.4.2305", uniqueID: "A.Mod", version: "1.0.0", name: "Alpha"})
	rep, err := s.Report(nil)
	if err != nil {
		t.Fatal(err)
	}
	got := rep["lethal-company"]
	if len(got.Duplicates) != 0 {
		t.Fatalf("loader items grouped as duplicates: %v", got.Duplicates)
	}
	for _, u := range got.Unused {
		if u.Name != "" || u.Version != "" {
			t.Fatalf("loader item %s named from a manifest inside it: %+v", u.Key, u)
		}
	}
}
