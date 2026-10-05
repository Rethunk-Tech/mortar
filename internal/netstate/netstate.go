// Package netstate remembers, per mod source, whether the last real request reached it, so the window can say a
// source is offline without polling.
package netstate

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"
)

// probeURL is a cheap address on each source's own host; any HTTP answer from it means the host is reachable.
var probeURL = map[string]string{
	"nexus":        "https://api.nexusmods.com",
	"github":       "https://api.github.com",
	"thunderstore": "https://thunderstore.io",
	"modrinth":     "https://api.modrinth.com",
	"itch":         "https://itch.io",
	"moddrop":      "https://www.moddrop.com",
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
}

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
	defer mu.Unlock()
	st := states[id]
	st.ID = id
	if err == nil {
		st.Unreachable, st.LastOK = false, time.Now()
	} else {
		st.Unreachable, st.LastFail = true, time.Now()
	}
	states[id] = st
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
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
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
