package devsvc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestUploadGitHubCreatesTheReleaseOrAddsToTheExistingOne(t *testing.T) {
	old := gh.DefaultAuth
	gh.DefaultAuth = &gh.Auth{Run: func(context.Context) ([]byte, error) { return []byte("tok"), nil }}
	t.Cleanup(func() { gh.DefaultAuth = old })

	var srv *httptest.Server
	var created map[string]any
	var uploads []string
	released := false
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/Ab/shop-drop/releases/tags/v1.0.0":
			if !released {
				http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte(`{"id":7,"tag_name":"v1.0.0","html_url":"page","upload_url":"` + srv.URL + `/up/7/assets{?name}","assets":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/Ab/shop-drop/releases":
			_ = json.NewDecoder(r.Body).Decode(&created)
			released = true
			_, _ = w.Write([]byte(`{"id":7,"tag_name":"v1.0.0","html_url":"page","upload_url":"` + srv.URL + `/up/7/assets{?name}"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/up/7/assets":
			uploads = append(uploads, r.URL.Query().Get("name"))
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":9}`))
		default:
			http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	writePackedZip(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("## 1.0.0\n\n- first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tagged := true
	svc.githubAPI = srv.URL
	svc.HTTP = srv.Client()
	svc.Run = func(_ context.Context, _, name string, args ...string) ([]byte, error) {
		switch {
		case name == "git" && args[0] == "remote":
			return []byte("git@github.com:Ab/shop-drop.git\n"), nil
		case name == "git" && args[0] == "ls-remote" && tagged:
			return []byte("abc\trefs/tags/v1.0.0\n"), nil
		}
		return nil, nil
	}
	ctx := t.Context()

	tagged = false
	if _, err := svc.UploadGitHub(ctx, "lethal-company", dir, "1.0.0"); usererr.KindOf(err) != usererr.Invalid || !strings.Contains(err.Error(), "v1.0.0") {
		t.Fatalf("untagged: %v", err)
	}
	tagged = true
	url, err := svc.UploadGitHub(ctx, "lethal-company", dir, "1.0.0")
	if err != nil || url != "page" {
		t.Fatalf("create = %q, %v", url, err)
	}
	if created["tag_name"] != "v1.0.0" || created["body"] != "- first" || len(uploads) != 1 || uploads[0] != "Ab-ShopDrop-1.0.0.zip" {
		t.Fatalf("created %v, uploads %v", created, uploads)
	}
	if _, err := svc.UploadGitHub(ctx, "lethal-company", dir, "1.0.0"); err != nil || len(uploads) != 2 {
		t.Fatalf("already released: %v, uploads %v", err, uploads)
	}

	gh.DefaultAuth = &gh.Auth{}
	if _, err := svc.UploadGitHub(ctx, "lethal-company", dir, "1.0.0"); usererr.KindOf(err) != usererr.Permission {
		t.Errorf("no token: %v", err)
	}
}
