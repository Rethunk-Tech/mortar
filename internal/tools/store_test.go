package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestStoreRoundTripPerGame(t *testing.T) {
	t.Parallel()
	st := &Store{root: t.TempDir()}

	got, err := st.List("stardew")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("absent file = %#v, %v; want empty non-nil", got, err)
	}

	a, err := st.Add("stardew", Tool{Name: "A", Executable: "/bin/a", Arguments: []string{"-x"}})
	if err != nil || a.ID == "" {
		t.Fatalf("Add = %+v, %v", a, err)
	}
	if _, err := st.Add("lethal-company", Tool{Name: "B", Executable: "/bin/b"}); err != nil {
		t.Fatal(err)
	}
	if list, _ := st.List("stardew"); len(list) != 1 || list[0].Name != "A" || list[0].Arguments[0] != "-x" {
		t.Fatalf("stardew list = %+v", list)
	}
	if list, _ := st.List("lethal-company"); len(list) != 1 || list[0].Name != "B" {
		t.Fatalf("lethal-company list = %+v", list)
	}

	a.Name = "A2"
	if err := st.Update("stardew", a); err != nil {
		t.Fatal(err)
	}
	if list, _ := st.List("stardew"); list[0].Name != "A2" {
		t.Fatalf("after Update = %+v", list)
	}
	if err := st.Remove("stardew", a.ID); err != nil {
		t.Fatal(err)
	}
	list, err := st.List("stardew")
	if err != nil || list == nil || len(list) != 0 {
		t.Fatalf("after Remove = %#v, %v; want empty non-nil", list, err)
	}
	if list, _ := st.List("lethal-company"); len(list) != 1 {
		t.Fatalf("removing from one game touched another: %+v", list)
	}
}

func TestStoreUnknownAndEmptyIDs(t *testing.T) {
	t.Parallel()
	st := &Store{root: t.TempDir()}
	if err := st.Update("stardew", Tool{ID: "nope", Name: "x"}); usererr.KindOf(err) != usererr.NotFound {
		t.Errorf("Update unknown id: %v", err)
	}
	if err := st.Remove("stardew", "nope"); usererr.KindOf(err) != usererr.NotFound {
		t.Errorf("Remove unknown id: %v", err)
	}
	if err := st.Update("stardew", Tool{Name: "x"}); err == nil || usererr.KindOf(err) == usererr.NotFound {
		t.Errorf("Update empty id: %v", err)
	}
	if err := st.Remove("stardew", ""); err == nil {
		t.Error("Remove empty id succeeded")
	}
	if _, err := os.Stat(filepath.Join(st.root, "stardew.json")); err == nil {
		t.Error("a refused change wrote the file")
	}
}

func TestLaunchUnknownToolIsNotFound(t *testing.T) {
	t.Parallel()
	s := &Service{store: &Store{root: t.TempDir()}}
	if err := s.Launch("stardew", "p1", "nope"); usererr.KindOf(err) != usererr.NotFound {
		t.Fatalf("Launch unknown id: %v", err)
	}
}
