package modrinth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStandingsFlagsArchivedProjectsByIDOrSlug(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects" || r.URL.Query().Get("ids") != `["AANobbMI","fine","old-mod"]` {
			t.Errorf("request %s", r.URL)
		}
		_, _ = w.Write([]byte(`[{"id":"AANobbMI","slug":"sodium","status":"archived"},{"id":"zzz","slug":"fine","status":"approved"},{"id":"qqq","slug":"old-mod","status":"archived"}]`))
	}))
	t.Cleanup(srv.Close)
	got, err := Driver{URL: srv.URL}.Standings(context.Background(), "1", []string{"AANobbMI", "fine", "old-mod"})
	if err != nil || len(got) != 2 || got["AANobbMI"].State != "archived" || got["old-mod"].State != "archived" {
		t.Fatalf("%+v %v", got, err)
	}
}
