package source

import (
	"errors"
	"net/http"
	"strings"
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
	e := Busy("Modrinth", &http.Response{Header: http.Header{"Retry-After": {"120"}}})
	if d := time.Until(e.Reset); d < 110*time.Second || d > 120*time.Second || !errors.Is(e, ErrBusy) {
		t.Errorf("reset %v", e.Reset)
	}
}

func TestQueryLen(t *testing.T) {
	if (Query{}).Len() != PageSize || (Query{Size: 7}).Len() != 7 {
		t.Fatal("Len should use Size when set, else PageSize")
	}
}

func TestReleased(t *testing.T) {
	if Released("not-a-time") != 0 {
		t.Fatal("garbage must not parse")
	}
	if Released("2026-01-01T00:00:00Z") == 0 {
		t.Fatal("RFC3339 must parse")
	}
}

func TestCategoryMatchAndUniqueNames(t *testing.T) {
	if !CategoryMatch([]string{"Mod"}, nil, nil) || CategoryMatch([]string{"Mod"}, []string{"Map"}, nil) ||
		!CategoryMatch([]string{"Mod"}, []string{"mod"}, nil) || CategoryMatch([]string{"Mod"}, nil, []string{"mod"}) {
		t.Fatal("include and exclude are case-insensitive; no include list lets every mod through")
	}
	if got := UniqueNames([]string{"", "A", "a", " B "}); strings.Join(got, ",") != "A,B" {
		t.Fatalf("UniqueNames = %v", got)
	}
}
