package devsvc

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// trashableProject is a created project in a folder whose name has a space and a "²", as players' folders do.
func trashableProject(t *testing.T) (*Service, string) {
	t.Helper()
	svc, _ := testService(t, testTemplate)
	parent := filepath.Join(t.TempDir(), "Jogos² Teste")
	if err := os.MkdirAll(parent, 0o750); err != nil {
		t.Fatal(err)
	}
	p, err := svc.Create(t.Context(), NewMod{
		Game: "lethal-company", Name: "CadaversRedone", Parent: parent, Repo: "CadaversRedone",
		Values: map[string]string{"description": "d", "team": "Ab"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, p.Dir
}

// Mortar's own watches and builds are let go of before the folder moves, and the project is forgotten only once it has.
func TestTrashLetsGoOfTheProjectBeforeMovingItAndForgetsItOnlyOnceMoved(t *testing.T) {
	svc, dir := trashableProject(t)
	type hold struct{ folderThere, listed, buildHeld, released, listedAtRelease bool }
	var holds []*hold
	svc.Unwatch = func(_ context.Context, held string) func() {
		h := &hold{folderThere: safefs.IsDir(held), listed: len(svc.Projects("lethal-company")) == 1}
		if mu, ok := building.Load(held); ok {
			if m, ok := mu.(*sync.Mutex); ok && !m.TryLock() {
				h.buildHeld = true
			} else if ok {
				m.Unlock()
			}
		}
		holds = append(holds, h)
		return func() { h.released, h.listedAtRelease = true, len(svc.Projects("lethal-company")) == 1 }
	}

	if runtime.GOOS != "windows" {
		// A trash folder that cannot be made fails the move; Windows has its own test with a held folder.
		notADir := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(notADir, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("XDG_DATA_HOME", notADir)
		if err := svc.Trash("lethal-company", dir); err == nil || len(svc.Projects("lethal-company")) != 1 || !safefs.IsDir(dir) {
			t.Fatalf("a failed move: err %v, projects %+v", err, svc.Projects("lethal-company"))
		}
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := svc.Trash("lethal-company", dir); err != nil || len(svc.Projects("lethal-company")) != 0 {
		t.Fatalf("trash: %v, projects %+v", err, svc.Projects("lethal-company"))
	}
	if want := map[bool]int{true: 1, false: 2}[runtime.GOOS == "windows"]; len(holds) != want {
		t.Fatalf("the watch was held %d times, want once per Trash", len(holds))
	}
	for i, h := range holds {
		if !h.folderThere || !h.listed || !h.buildHeld || !h.released {
			t.Errorf("hold %d: %+v; want it taken before the move, with the build lock, and released", i, *h)
		}
	}
	if holds[len(holds)-1].listedAtRelease || len(holds) == 2 && !holds[0].listedAtRelease {
		t.Errorf("listed at release: %+v; want kept after a failed move, forgotten before the watch resumes after one", holds)
	}
}

func TestInUseCarriesTheHoldersForTheGUI(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "CadaversRedone")
	err := inUse(dir, []string{"Visual Studio Code", "dotnet"})
	if usererr.KindOf(err) != usererr.InUse {
		t.Fatalf("kind %v", usererr.KindOf(err))
	}
	if got := string(usererr.Marshal(err)); got != `{"kind":"in_use","detail":{"folder":`+strconv.Quote(dir)+`,"holders":["Visual Studio Code","dotnet"]}}` {
		t.Errorf("wire cause %s", got)
	}
	if got := inUse(dir, []string{"dotnet"}).Error(); !strings.HasSuffix(got, "CadaversRedone is in use by: dotnet") {
		t.Errorf("text %q", got)
	}
	got := inUse(dir, nil).Error()
	if !strings.Contains(got, dir) || !strings.Contains(got, "Windows denied access") || !strings.Contains(got, "a terminal open in it") {
		t.Errorf("nothing found: %q", got)
	}
	if !strings.Contains(string(usererr.Marshal(inUse(dir, nil))), `"holders":[]`) {
		t.Error("an unknown holder list must marshal as an empty array")
	}
}
