package devsvc

import (
	"crypto/md5" // #nosec G501 -- checks the Content-MD5 the upload sends
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// A fake Thunderstore that splits a 10-byte package in two parts, checks each part's bytes and MD5, and records the
// finish and submit bodies.
func TestThunderstoreUploadFollowsTheCLIProtocol(t *testing.T) {
	pkg := filepath.Join(t.TempDir(), "Ab-ShopDrop-0.1.0.zip")
	if err := fsx.WriteFile(pkg, []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	var srv *httptest.Server
	var finished, submitted map[string]any
	var auth []string
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.Method == http.MethodPut:
			sum := md5.Sum(body) // #nosec G401 -- the header under test
			want := map[string]string{"/part/1": "01234", "/part/2": "56789"}[r.URL.Path]
			if string(body) != want || r.Header.Get("Content-MD5") != base64.StdEncoding.EncodeToString(sum[:]) {
				t.Errorf("part %s = %q", r.URL.Path, body)
			}
			w.Header().Set("ETag", `"`+strings.TrimPrefix(r.URL.Path, "/part/")+`"`)
			return
		case strings.HasSuffix(r.URL.Path, "/initiate-upload/"):
			auth = append(auth, r.Header.Get("Authorization"))
			_, _ = fmt.Fprintf(w, `{"user_media":{"uuid":"u1"},"upload_urls":[{"part_number":1,"url":"%[1]s/part/1","offset":0,"length":5},{"part_number":2,"url":"%[1]s/part/2","offset":5,"length":5}]}`, srv.URL)
		case strings.HasSuffix(r.URL.Path, "/usermedia/u1/finish-upload/"):
			_ = json.Unmarshal(body, &finished)
		case strings.HasSuffix(r.URL.Path, "/submission/submit/"):
			auth = append(auth, r.Header.Get("Authorization"))
			_ = json.Unmarshal(body, &submitted)
			_, _ = w.Write([]byte(`{"available_communities":[{"url":"https://thunderstore.io/c/lethal-company/p/Ab/ShopDrop/"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	ts := thunderstoreAPI{client: srv.Client(), site: srv.URL + "/", token: "tok"}
	uuid, err := ts.uploadFile(t.Context(), pkg)
	if err != nil || uuid != "u1" {
		t.Fatalf("upload = %q, %v", uuid, err)
	}
	if got, _ := json.Marshal(finished["parts"]); string(got) != `[{"ETag":"\"1\"","PartNumber":1},{"ETag":"\"2\"","PartNumber":2}]` {
		t.Errorf("finished parts = %s", got)
	}
	url, err := ts.submit(t.Context(), uuid, "Ab", "lethal-company", []string{"mods"})
	if err != nil || url != "https://thunderstore.io/c/lethal-company/p/Ab/ShopDrop/" {
		t.Fatalf("submit = %q, %v", url, err)
	}
	if submitted["author_name"] != "Ab" || submitted["upload_uuid"] != "u1" || fmt.Sprint(submitted["community_categories"]) != "map[lethal-company:[mods]]" {
		t.Errorf("submitted = %v", submitted)
	}
	if !slices.Equal(auth, []string{"Bearer tok", "Bearer tok"}) {
		t.Errorf("auth = %q", auth)
	}
}
