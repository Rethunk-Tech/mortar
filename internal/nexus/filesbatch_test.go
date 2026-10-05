package nexus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFilesOfAsksForEveryModInOneRequest(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body struct {
			Query string `json:"query"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !strings.Contains(body.Query, "m41150: modFiles(modId: 41150") || !strings.Contains(body.Query, "m28261: modFiles(modId: 28261") {
			t.Errorf("query %q", body.Query)
		}
		_, _ = w.Write([]byte(`{"data":{"m41150":[{"fileId":185334,"name":"Quest Helper","version":"0.3.4","category":"MAIN"}],"m28261":[]}}`))
	}))
	defer srv.Close()
	c := New("test").WithKey("k")
	c.BaseURL = srv.URL
	got, err := c.FilesOf(context.Background(), stardew, []int{41150, 28261})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(got[41150]) != 1 || got[41150][0].FileID != 185334 {
		t.Fatalf("calls %d files %+v", calls, got)
	}
}
