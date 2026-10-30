package nexus

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-Tech/go-nexusmods"
	"github.com/Rethunk-Tech/mortar/internal/netstate"
)

func nexusState(t *testing.T) (netstate.State, bool) {
	t.Helper()
	for _, st := range (&netstate.Service{}).States() {
		if st.ID == "nexus" {
			return st, true
		}
	}
	return netstate.State{}, false
}

// An answer of any status means Nexus was reached; a connection that fails means it was not.
func TestObservedHTTPTellsNetstateWhetherNexusAnswered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	c := nexusmods.New("Mortar", "1")
	c.BaseURL, c.HTTP = srv.URL, ObservedHTTP(http.DefaultTransport)
	c = c.WithKey("k")

	_, _ = c.Validate(t.Context())
	if st, ok := nexusState(t); !ok || st.Unreachable {
		t.Fatalf("a 429 answer marked Nexus unreachable: %+v", st)
	}
	srv.Close()
	_, _ = c.Validate(t.Context())
	if st, ok := nexusState(t); !ok || !st.Unreachable {
		t.Fatalf("a refused connection left Nexus reachable: %+v", st)
	}
}
