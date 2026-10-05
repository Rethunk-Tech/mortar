package netstate

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecordKeepsStateOnNonNetworkErrors(t *testing.T) {
	states = map[string]State{}
	Record("nexus", context.DeadlineExceeded)
	Record("nexus", errors.New("401 unauthorized"))
	Record("nexus", context.Canceled)
	if got := (&Service{}).States(); len(got) != 1 || !got[0].Unreachable {
		t.Fatalf("want one unreachable source, got %+v", got)
	}
	Record("nexus", nil)
	if got := (&Service{}).States(); got[0].Unreachable || got[0].LastOK.IsZero() || got[0].LastFail.IsZero() {
		t.Fatalf("success should clear Unreachable and keep both times, got %+v", got[0])
	}
}

func TestRecheckProbesOnlyUnreachableSources(t *testing.T) {
	states = map[string]State{}
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	probeURL["test"] = srv.URL
	defer delete(probeURL, "test")
	Record("test", nil)
	(&Service{}).Recheck(context.Background())
	if hits != 0 {
		t.Fatalf("a reachable source was probed")
	}
	Record("test", context.DeadlineExceeded)
	got := (&Service{}).Recheck(context.Background())
	if hits != 1 || got[0].Unreachable {
		t.Fatalf("an HTTP answer, even 403, should clear the failure: hits=%d %+v", hits, got)
	}
}

func TestOnChangeFiresOnlyWhenReachabilityFlips(t *testing.T) {
	states = map[string]State{}
	var calls int
	OnChange = func() { calls++ }
	defer func() { OnChange = nil }()
	Record("nexus", nil)
	Record("nexus", nil)
	Record("nexus", context.DeadlineExceeded)
	Record("nexus", context.DeadlineExceeded)
	Record("nexus", nil)
	if calls != 3 {
		t.Fatalf("OnChange called %d times, want 3 (first sight, down, up)", calls)
	}
}
