// Package netstate remembers, per mod source, whether the last real request reached it, so the window can say a
// source is offline without polling.
package netstate

import (
	"context"
	"errors"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// probeURL is a cheap address on each source's own host; any HTTP answer from it means the host is reachable.
var probeURL = map[string]string{
	"nexus":        "https://api.nexusmods.com",
	"github":       "https://api.github.com",
	"thunderstore": "https://thunderstore.io",
	"modrinth":     "https://api.modrinth.com",
	"curseforge":   "https://api.curseforge.com",
	"itch":         "https://itch.io",
}

const probeTimeout = 10 * time.Second

// State is what the last requests to one source showed.
type State struct {
	ID string `json:"id"`
	// Unreachable is true when the last real request failed with a network error and nothing has succeeded since.
	Unreachable bool `json:"unreachable"`
	// LastOK is when a request to the source last succeeded; zero when none has this session.
	LastOK time.Time `json:"lastOK"`
	// LastFail is when the last network error happened; zero when there has been none.
	LastFail time.Time `json:"lastFail"`
	// LastError is the network error behind LastFail; empty once a request succeeds.
	LastError string `json:"lastError"`
	// LastReason sorts LastError for the window to put in its own words: ReasonDNS, ReasonTimeout, ReasonRefused or
	// ReasonOther; empty once a request succeeds.
	LastReason string `json:"lastReason"`
}

// The reasons a source could not be reached.
const (
	ReasonDNS     = "dns"
	ReasonTimeout = "timeout"
	ReasonRefused = "refused"
	ReasonOther   = "other"
)

func reasonOf(err error) string {
	if _, ok := errors.AsType[*net.DNSError](err); ok {
		return ReasonDNS
	}
	if errors.Is(err, refusedErrno) {
		return ReasonRefused
	}
	ne, ok := errors.AsType[net.Error](err)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, timedOutErrno) || (ok && ne.Timeout()) {
		return ReasonTimeout
	}
	return ReasonOther
}

// OnChange is called, when set, after a source flips between reachable and unreachable, so the window need not poll.
var OnChange func()

var (
	mu     sync.Mutex
	states = map[string]State{}
	// Client sends the probes; tests replace it.
	client = http.DefaultClient
)

// isNetwork reports whether err means the host could not be reached (DNS, connection, timeout), not that it answered
// badly.
func isNetwork(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	var ne net.Error
	return errors.Is(err, context.DeadlineExceeded) || errors.As(err, &ne)
}

// Record notes the outcome of a real request to source id. A nil err is a success; a network error marks the source
// unreachable; any other error (the source answered, but badly) leaves its state alone.
func Record(id string, err error) {
	if err != nil && !isNetwork(err) {
		return
	}
	mu.Lock()
	st := states[id]
	was, seen := st.Unreachable, st.ID != ""
	st.ID = id
	if err == nil {
		st.Unreachable, st.LastOK, st.LastError, st.LastReason = false, time.Now(), "", ""
	} else {
		st.Unreachable, st.LastFail, st.LastError, st.LastReason = true, time.Now(), err.Error(), reasonOf(err)
	}
	states[id] = st
	mu.Unlock()
	if OnChange != nil && (st.Unreachable != was || !seen) {
		OnChange()
	}
}

// Service is the window's reachability API.
type Service struct{}

// States lists every source that has been requested this session, by id.
func (*Service) States() []State {
	mu.Lock()
	defer mu.Unlock()
	out := make([]State, 0, len(states))
	for _, st := range states {
		out = append(out, st)
	}
	slices.SortFunc(out, func(a, b State) int { return strings.Compare(a.ID, b.ID) })
	return out
}

// Retry probes source id now and returns every state.
func (s *Service) Retry(ctx context.Context, id string) []State {
	probe(ctx, id)
	return s.States()
}

// Recheck probes each source that is unreachable; the window calls it every few minutes while focused, so a source
// that is fine costs no request.
func (s *Service) Recheck(ctx context.Context) []State {
	for _, st := range s.States() {
		if st.Unreachable {
			probe(ctx, st.ID)
		}
	}
	return s.States()
}

func probe(ctx context.Context, id string) {
	url, ok := probeURL[id]
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return
	}
	resp, err := client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
	Record(id, err)
}
