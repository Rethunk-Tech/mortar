package sharesvc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseShortRoute(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		value string
		want  shortRoute
		ok    bool
	}{
		{
			name:  "https",
			value: "https://mortar.rethunk.tech/stardew/s/AbCdEfGh12",
			want:  shortRoute{game: "stardew", code: "AbCdEfGh12"},
			ok:    true,
		},
		{
			name:  "mortar scheme",
			value: "mortar://stardew/s/AbCdEfGh12",
			want:  shortRoute{game: "stardew", code: "AbCdEfGh12"},
			ok:    true,
		},
		{
			name:  "versioned https",
			value: "https://mortar.rethunk.tech/stardew/s/AbCdEfGh12/3",
			want:  shortRoute{game: "stardew", code: "AbCdEfGh12", version: 3},
			ok:    true,
		},
		{
			name:  "versioned mortar",
			value: "mortar://stardew/s/AbCdEfGh12/3",
			want:  shortRoute{game: "stardew", code: "AbCdEfGh12", version: 3},
			ok:    true,
		},
		{
			name:  "payload page is not a short code",
			value: "https://mortar.rethunk.tech/stardew/p/#abc",
		},
		{
			name:  "bad code charset",
			value: "mortar://stardew/s/AbCd-EfGh1",
		},
		{
			name:  "wrong host",
			value: "https://example.com/stardew/s/AbCdEfGh12",
		},
		{
			name:  "code too short",
			value: "mortar://stardew/s/AbCdEfGh1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseShortRoute(tc.value)
			if ok != tc.ok {
				t.Fatalf("parseShortRoute(%q) ok=%v want %v", tc.value, ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("parseShortRoute(%q) = %+v want %+v", tc.value, got, tc.want)
			}
		})
	}
}

func TestResolveShortShare(t *testing.T) {
	const payload = "dGVzdA"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/shares/")
		switch rest {
		case "FoundCode1":
			writeShareJSON(t, w, http.StatusOK, shareAPIResponse{
				Code: "FoundCode1", Version: 1, Game: "stardew", Payload: payload,
			})
		case "FoundCode1/2":
			writeShareJSON(t, w, http.StatusOK, shareAPIResponse{
				Code: "FoundCode1", Version: 2, Game: "stardew", Payload: payload,
			})
		case "GoneCode12":
			http.Error(w, "not found", http.StatusNotFound)
		case "BoomCode12":
			http.Error(w, "broken", http.StatusInternalServerError)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	prev := apiBaseURL
	apiBaseURL = srv.URL
	t.Cleanup(func() { apiBaseURL = prev })

	cases := []struct {
		name    string
		value   string
		want    string
		wantErr string
	}{
		{name: "found", value: "mortar://stardew/s/FoundCode1", want: payload},
		{name: "versioned", value: "https://mortar.rethunk.tech/stardew/s/FoundCode1/2", want: payload},
		{name: "unknown", value: "mortar://stardew/s/GoneCode12", wantErr: "unknown"},
		{name: "server error", value: "mortar://stardew/s/BoomCode12", wantErr: "offline"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			route, ok := parseShortRoute(tc.value)
			if !ok {
				t.Fatalf("parseShortRoute(%q) failed", tc.value)
			}
			got, err := fetchSharePayload(t.Context(), route)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("fetchSharePayload(%q) err=nil want %q", tc.value, tc.wantErr)
				}
				if !strings.Contains(strings.ToLower(err.Error()), tc.wantErr) {
					t.Fatalf("fetchSharePayload(%q) err=%v want substring %q", tc.value, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("fetchSharePayload(%q): %v", tc.value, err)
			}
			if got != tc.want {
				t.Fatalf("fetchSharePayload(%q) payload=%q want %q", tc.value, got, tc.want)
			}
		})
	}
}

func writeShareJSON(t *testing.T, w http.ResponseWriter, status int, body shareAPIResponse) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatal(err)
	}
}
