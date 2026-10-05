package source

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

type fakeSource struct {
	Source
	id string
}

func (f fakeSource) ID() string { return f.id }

func TestRegistryContract(t *testing.T) {
	Register(fakeSource{id: "contract-a"})
	Register(fakeSource{id: "contract-b"})
	for id, want := range map[string]bool{"contract-a": true, "CONTRACT-B": true, "contract-none": false} {
		if _, ok := Get(id); ok != want {
			t.Errorf("Get(%q) = %v, want %v", id, ok, want)
		}
	}
	g := components.GameInfo{Sources: []components.GameSource{{ID: "contract-b"}, {ID: "contract-none"}, {ID: "contract-a"}}}
	got := ForGame(g)
	if len(got) != 2 || got[0].ID() != "contract-b" || got[1].ID() != "contract-a" {
		t.Errorf("ForGame lists catalog order and skips a source with no driver: %v", got)
	}
}

func TestBusyKeepsTheRetryAfterWait(t *testing.T) {
	at := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	for v, ok := range map[string]func(time.Time) bool{
		"120":                      func(r time.Time) bool { return time.Until(r) > 110*time.Second && time.Until(r) <= 120*time.Second },
		at.Format(http.TimeFormat): func(r time.Time) bool { return r.Equal(at) },
		"":                         time.Time.IsZero,
	} {
		e := Busy("Modrinth", &http.Response{Header: http.Header{"Retry-After": {v}}})
		if !ok(e.Reset) || !errors.Is(e, ErrBusy) {
			t.Errorf("Retry-After %q: reset %v", v, e.Reset)
		}
	}
}
