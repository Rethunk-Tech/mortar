package queue

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

func TestTopPausesTheLastRunningDownloadAndItResumesAfter(t *testing.T) {
	cdn := &blockingCDN{release: make(chan struct{})}
	srv := httptest.NewServer(cdn)
	defer srv.Close()
	f := packageFixture(t, srv.URL)
	f.start()
	addPackages(t, f, "A", "B", "C", "D")
	f.wait("three downloading", func(State) bool { return cdn.has("A") && cdn.has("B") && cdn.has("C") })

	f.s.Top(f.stateOf("Me-D").ID)
	f.wait("D started, C yielded", func(State) bool { return cdn.has("D") && f.stateOf("Me-C").State == StatePaused })
	if c := f.stateOf("Me-C"); !c.AutoResume {
		t.Fatalf("C was paused as a user pause: %+v", c)
	}
	if st := f.s.State(); st.Items[0].Package != "Me-D" {
		t.Fatalf("D is not first: %s", st.Items[0].Package)
	}
	close(cdn.release)
	f.wait("all done", func(st State) bool {
		for _, it := range st.Items {
			if it.State != StateDone {
				return false
			}
		}
		return len(st.Items) == 4
	})
}

func TestTopMovesPendingDependenciesAhead(t *testing.T) {
	f := newFixture(t)
	f.s.d.Closure = func(_ context.Context, _ string, roots []thunderstore.Ref, _ bool) ([]thunderstore.Resolved, error) {
		out := []thunderstore.Resolved{}
		for _, r := range roots {
			if r.Name == "Mod" {
				out = append(out, thunderstore.Resolved{Namespace: "Me", Name: "Lib", Version: "1.0.0", URL: "https://x/Lib"})
				out = append(out, thunderstore.Resolved{Namespace: r.Namespace, Name: r.Name, Version: "1.0.0", URL: "https://x/Mod", Dependencies: []string{"Me-Lib-1.0.0"}})
				continue
			}
			out = append(out, thunderstore.Resolved{Namespace: r.Namespace, Name: r.Name, Version: "1.0.0", URL: "https://x/" + r.Name})
		}
		return out, nil
	}
	addPackages(t, f, "Other", "Mod")
	f.s.PauseItem(f.stateOf("Me-Lib").ID)
	f.s.Top(f.stateOf("Me-Mod").ID)
	var order []string
	for _, it := range f.s.State().Items {
		order = append(order, it.Package+":"+it.State)
	}
	if got := strings.Join(order, " "); got != "Me-Lib:queued Me-Mod:queued Me-Other:queued" {
		t.Fatalf("order %s", got)
	}
}

func TestQueueOrderSurvivesAReload(t *testing.T) {
	f := newFixture(t)
	f.s.d.Closure = func(_ context.Context, _ string, roots []thunderstore.Ref, _ bool) ([]thunderstore.Resolved, error) {
		var out []thunderstore.Resolved
		for _, r := range roots {
			out = append(out, thunderstore.Resolved{Namespace: r.Namespace, Name: r.Name, Version: "1.0.0", URL: "https://x/" + r.Name})
		}
		return out, nil
	}
	f.s.Pause()
	addPackages(t, f, "A", "B", "C")
	f.s.Top(f.stateOf("Me-C").ID)
	f.s.Move(f.stateOf("Me-C").ID, 1)

	again, err := New(f.s.d)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, it := range again.State().Items {
		got = append(got, fmt.Sprintf("%s:%s:%v", it.Package, it.State, it.AutoResume))
	}
	if want := "Me-A:queued:false Me-C:queued:false Me-B:queued:false"; strings.Join(got, " ") != want {
		t.Fatalf("got %v want %s", got, want)
	}
}
