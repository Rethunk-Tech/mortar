package curseforge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStandingsFlagsInactiveAbandonedDeletedAndUnavailableMods(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ModIDs []int `json:"modIds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || r.Method != http.MethodPost || r.URL.Path != "/mods" || len(body.ModIDs) != 6 {
			t.Errorf("request %s %s %+v %v", r.Method, r.URL, body, err)
		}
		_, _ = w.Write([]byte(`{"data":[
			{"id":1,"status":4,"isAvailable":true},
			{"id":2,"status":7,"isAvailable":true},
			{"id":3,"status":8,"isAvailable":true},
			{"id":4,"status":9,"isAvailable":true},
			{"id":5,"status":4,"isAvailable":false},
			{"id":6,"status":4}]}`))
	}))
	t.Cleanup(srv.Close)
	d := Driver{URL: srv.URL, Key: func() string { return "k" }}
	got, err := d.Standings(context.Background(), "1", []string{"1", "2", "3", "4", "5", "6", "name"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"2": "inactive", "3": "abandoned", "4": "removed", "5": "unavailable"}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for id, state := range want {
		if got[id].State != state {
			t.Errorf("mod %s = %q, want %q", id, got[id].State, state)
		}
	}
}
