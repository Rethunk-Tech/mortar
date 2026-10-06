package netstate

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"syscall"
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

func TestRecordKeepsTheLastErrorUntilASuccess(t *testing.T) {
	Record("thunderstore", &net.DNSError{Err: "no such host", Name: "thunderstore.io"})
	st := (&Service{}).States()
	var got State
	for _, s := range st {
		if s.ID == "thunderstore" {
			got = s
		}
	}
	if !got.Unreachable || got.LastError == "" {
		t.Fatalf("after failure %+v", got)
	}
	Record("thunderstore", nil)
	for _, s := range (&Service{}).States() {
		if s.ID == "thunderstore" && (s.Unreachable || s.LastError != "") {
			t.Fatalf("after success %+v", s)
		}
	}
}

func TestRecordSortsWhyASourceCouldNotBeReached(t *testing.T) {
	refused := &net.OpError{Op: "proxyconnect", Net: "tcp", Err: &os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED}}
	for err, want := range map[error]string{
		&url.Error{Op: "Get", URL: "https://api.nexusmods.com", Err: refused}:                         ReasonRefused,
		&url.Error{Op: "Get", URL: "https://api.github.com", Err: &net.DNSError{Err: "no such host"}}: ReasonDNS,
		context.DeadlineExceeded: ReasonTimeout,
		&net.OpError{Op: "read", Net: "tcp", Err: errors.New("connection reset by peer")}: ReasonOther,
	} {
		states = map[string]State{}
		Record("nexus", err)
		if got := (&Service{}).States()[0].LastReason; got != want {
			t.Fatalf("%v: reason %q, want %q", err, got, want)
		}
	}
	Record("nexus", nil)
	if got := (&Service{}).States()[0].LastReason; got != "" {
		t.Fatalf("a success kept reason %q", got)
	}
}
