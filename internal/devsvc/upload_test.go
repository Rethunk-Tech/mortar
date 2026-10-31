package devsvc

import (
	"crypto/md5" // #nosec G501 -- checks the Content-MD5 the upload sends
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/zalando/go-keyring"
)

// A fake Thunderstore that splits a 10-byte package in two parts, checks each part's bytes and MD5, and records the
// finish and submit bodies.
func TestThunderstoreUploadFollowsTheCLIProtocol(t *testing.T) {
	pkg := filepath.Join(t.TempDir(), "Ab-ShopDrop-0.1.0.zip")
	if err := safefs.WriteFile(pkg, []byte("0123456789"), 0o600); err != nil {
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

func TestUploadRefusesAnUntrustedProject(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	p := svc.Settings.Get().GamePrefs("lethal-company").DevProjects[0]
	next := p
	next.Untrusted = true
	if err := svc.Settings.ReplaceDevProject("lethal-company", p, next); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UploadThunderstore(t.Context(), "lethal-company", dir); usererr.KindOf(err) != usererr.Unconfirmed {
		t.Errorf("err = %v", err)
	}
}

func writePackage(t *testing.T, body string) string {
	t.Helper()
	pkg := filepath.Join(t.TempDir(), "Ab-ShopDrop-1.2.0.zip")
	if err := safefs.WriteFile(pkg, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return pkg
}

// A fake Nexus Mods v3 API and its storage. It records each request as "METHOD path" with its body, and answers the
// upload as "created" once before "available".
type fakeNexus struct {
	srv   *httptest.Server
	log   []string
	bodys map[string]string
	polls int
}

func newFakeNexus(t *testing.T, parts, partSize int) *fakeNexus {
	t.Helper()
	f := &fakeNexus{bodys: map[string]string{}}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		key := r.Method + " " + r.URL.Path
		f.log = append(f.log, key)
		f.bodys[key] = string(body)
		isAPI := strings.HasPrefix(r.URL.Path, "/v3/")
		if isAPI && (r.Header.Get("Apikey") != "k" || r.Header.Get("Application-Name") != "Mortar" || r.Header.Get("Application-Version") != "9.9.9") {
			t.Errorf("%s headers = %v", key, r.Header)
		}
		if !isAPI && r.Header.Get("Apikey") != "" {
			t.Errorf("%s leaked the API key to storage", key)
		}
		switch {
		case key == "POST /v3/uploads":
			_, _ = fmt.Fprintf(w, `{"data":{"id":"up1","state":"created","presigned_url":"%s/s3/whole"}}`, f.srv.URL)
		case key == "POST /v3/uploads/multipart":
			urls := make([]string, parts)
			for i := range urls {
				urls[i] = fmt.Sprintf("%s/s3/part%d", f.srv.URL, i+1)
			}
			raw, _ := json.Marshal(urls)
			_, _ = fmt.Fprintf(w, `{"data":{"id":"up1","state":"created","part_size_bytes":%d,"part_presigned_urls":%s,"complete_presigned_url":"%s/s3/complete"}}`, partSize, raw, f.srv.URL)
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/s3/"):
			w.Header().Set("ETag", `"etag-`+strings.TrimPrefix(r.URL.Path, "/s3/")+`"`)
		case key == "POST /s3/complete":
		case key == "POST /v3/uploads/up1/finalise":
			_, _ = w.Write([]byte(`{"data":{"id":"up1","state":"created"}}`))
		case key == "GET /v3/uploads/up1":
			f.polls++
			state := "created"
			if f.polls > 1 {
				state = "available"
			}
			_, _ = fmt.Fprintf(w, `{"data":{"id":"up1","state":"%s"}}`, state)
		case key == "POST /v3/mod-files/77/versions" || key == "POST /v3/mods/55/changelogs":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeNexus) uploader(singleMax int64) nexusUploader {
	return nexusUploader{client: f.srv.Client(), base: f.srv.URL + "/v3", key: "k", version: "9.9.9", poll: time.Millisecond, singleMax: singleMax}
}

func TestNexusUploadSinglePutThenVersionAndChangelog(t *testing.T) {
	f := newFakeNexus(t, 0, 0)
	pkg := writePackage(t, "0123456789")
	err := f.uploader(0).upload(t.Context(), pkg, "ShopDrop", NexusIDs{ModID: "55", FileID: "77"}, "1.2.0", "- fixed it")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"POST /v3/uploads", "PUT /s3/whole", "POST /v3/uploads/up1/finalise", "GET /v3/uploads/up1", "GET /v3/uploads/up1",
		"POST /v3/mod-files/77/versions", "POST /v3/mods/55/changelogs",
	}
	if !slices.Equal(f.log, want) {
		t.Fatalf("requests = %q", f.log)
	}
	if f.bodys["PUT /s3/whole"] != "0123456789" {
		t.Errorf("stored bytes = %q", f.bodys["PUT /s3/whole"])
	}
	for key, want := range map[string]string{
		"POST /v3/uploads":               `{"filename":"Ab-ShopDrop-1.2.0.zip","size_bytes":10}`,
		"POST /v3/mod-files/77/versions": `{"archive_existing_file":true,"file_category":"main","name":"ShopDrop","upload_id":"up1","version":"1.2.0"}`,
		"POST /v3/mods/55/changelogs":    `{"changelog":"- fixed it","version":"1.2.0"}`,
	} {
		if f.bodys[key] != want {
			t.Errorf("%s body = %s", key, f.bodys[key])
		}
	}
}

func TestNexusUploadMultipartCompletesWithEachETag(t *testing.T) {
	f := newFakeNexus(t, 3, 4)
	pkg := writePackage(t, "0123456789")
	// No mod id: no changelog request.
	if err := f.uploader(5).upload(t.Context(), pkg, "ShopDrop", NexusIDs{FileID: "77"}, "1.2.0", "- fixed it"); err != nil {
		t.Fatal(err)
	}
	for part, want := range map[string]string{"part1": "0123", "part2": "4567", "part3": "89"} {
		if f.bodys["PUT /s3/"+part] != want {
			t.Errorf("%s = %q", part, f.bodys["PUT /s3/"+part])
		}
	}
	wantXML := "<CompleteMultipartUpload>\n" +
		"  <Part>\n    <PartNumber>1</PartNumber>\n    <ETag>etag-part1</ETag>\n  </Part>\n" +
		"  <Part>\n    <PartNumber>2</PartNumber>\n    <ETag>etag-part2</ETag>\n  </Part>\n" +
		"  <Part>\n    <PartNumber>3</PartNumber>\n    <ETag>etag-part3</ETag>\n  </Part>\n</CompleteMultipartUpload>"
	if f.bodys["POST /s3/complete"] != wantXML {
		t.Errorf("complete = %q", f.bodys["POST /s3/complete"])
	}
	if slices.Contains(f.log, "POST /v3/mods/55/changelogs") || f.log[0] != "POST /v3/uploads/multipart" {
		t.Errorf("requests = %q", f.log)
	}
}

func TestNexusUploadRefusesPartsThatMissTheFile(t *testing.T) {
	f := newFakeNexus(t, 2, 4)
	err := f.uploader(5).upload(t.Context(), writePackage(t, "0123456789"), "ShopDrop", NexusIDs{FileID: "77"}, "1.2.0", "")
	if err == nil || slices.Contains(f.log, "PUT /s3/part1") {
		t.Errorf("err = %v, requests = %q", err, f.log)
	}
}

func TestSiteErrorsAndStorageAddresses(t *testing.T) {
	for _, c := range []struct {
		status int
		kind   usererr.Kind
		busy   bool
	}{
		{http.StatusUnauthorized, usererr.Permission, false},
		{http.StatusForbidden, usererr.Permission, false},
		{http.StatusTooManyRequests, usererr.RateLimited, true},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(c.status) }))
		pkg := writePackage(t, "x")
		errs := map[string]error{
			"nexus": nexusUploader{client: srv.Client(), base: srv.URL, key: "k"}.upload(t.Context(), pkg, "n", NexusIDs{FileID: "1"}, "1.0.0", ""),
			"curseforge": func() error {
				_, err := curseForgeUploader{client: srv.Client(), base: srv.URL, token: "t"}.upload(t.Context(), pkg, 1, "n", "")
				return err
			}(),
		}
		for site, err := range errs {
			if usererr.KindOf(err) != c.kind || errors.Is(err, source.ErrBusy) != c.busy {
				t.Errorf("%s %d: kind %v, err %v", site, c.status, usererr.KindOf(err), err)
			}
		}
		srv.Close()
	}
	for raw, base := range map[string]string{
		"http://bucket.example/part": "https://api.nexusmods.com/v3",
		"http://other.example/part":  "http://127.0.0.1:1/v3",
		"/relative":                  "https://api.nexusmods.com/v3",
	} {
		if _, err := storageURL(raw, base); err == nil {
			t.Errorf("%s followed under %s", raw, base)
		}
	}
	if _, err := storageURL("https://bucket.example/part", "https://api.nexusmods.com/v3"); err != nil {
		t.Error(err)
	}
}

func TestCurseForgeUploadSendsMetadataAndFileAsAForm(t *testing.T) {
	var meta map[string]any
	var file, name, token, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, token = r.URL.Path, r.Header.Get("X-Api-Token")
		form, err := r.MultipartReader()
		if err != nil {
			t.Error(err)
			return
		}
		for {
			part, err := form.NextPart()
			if err != nil {
				break
			}
			b, _ := io.ReadAll(io.LimitReader(part, 1<<20))
			switch part.FormName() {
			case "metadata":
				_ = json.Unmarshal(b, &meta)
			case "file":
				file, name = string(b), part.FileName()
			}
		}
		_, _ = w.Write([]byte(`{"id":4242}`))
	}))
	defer srv.Close()
	id, err := curseForgeUploader{client: srv.Client(), base: srv.URL, token: "tok", version: "9.9.9"}.
		upload(t.Context(), writePackage(t, "0123456789"), 31, "ShopDrop 1.2.0", "- fixed it")
	if err != nil || id != 4242 {
		t.Fatalf("id = %d, %v", id, err)
	}
	if path != "/api/projects/31/upload-file" || token != "tok" || file != "0123456789" || name != "Ab-ShopDrop-1.2.0.zip" {
		t.Errorf("path %q, token %q, file %q (%q)", path, token, file, name)
	}
	if fmt.Sprint(meta) != "map[changelog:- fixed it changelogType:markdown displayName:ShopDrop 1.2.0 releaseType:release]" {
		t.Errorf("metadata = %v", meta)
	}
}

func TestChangelogOfTakesOneVersionsNotes(t *testing.T) {
	log := "# Changelog\n\n## 1.2.0\n\n- b\n- c\n\n## 1.1.0\n\n- a\n"
	for version, want := range map[string]string{"1.2.0": "- b\n- c", "1.1.0": "- a", "1.0.0": ""} {
		if got := changelogOf(log, version); got != want {
			t.Errorf("%s = %q", version, got)
		}
	}
}

func TestSiteIDsAreKeptInTheProjectsRecordAndOffered(t *testing.T) {
	keyring.MockInit()
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	configured := func() []string {
		sites, _ := svc.UploadSites("lethal-company", dir)
		ids := []string{}
		for _, s := range sites {
			ids = append(ids, s.ID+fmt.Sprint(s.Configured))
		}
		return ids
	}
	if got := configured(); !slices.Equal(got, []string{"thunderstoretrue", "nexusfalse", "curseforgefalse"}) {
		t.Fatalf("sites before ids = %v", got)
	}
	info, err := svc.Inspect(t.Context(), "lethal-company", dir)
	if err != nil {
		t.Fatal(err)
	}
	edit := ProjectEdit{
		Name: info.Name, Description: info.Description, Author: info.Author, Team: info.Team, IDPrefix: info.IDPrefix,
		Categories: info.Categories, TestWith: info.TestWith,
		Nexus: NexusIDs{ModID: "55", FileID: "77"}, CurseForge: CurseForgeIDs{ProjectID: 31},
	}
	got, err := svc.Edit(t.Context(), "lethal-company", dir, edit)
	if err != nil || got.Nexus != edit.Nexus || got.CurseForge != edit.CurseForge {
		t.Fatalf("edited = %+v, %v", got, err)
	}
	body, _ := safefs.ReadFile(filepath.Join(dir, ".mortar", "template.json"))
	var rec map[string]any
	_ = json.Unmarshal(body, &rec)
	if fmt.Sprint(rec["nexus"], rec["curseforge"]) != "map[fileId:77 modId:55] map[projectId:31]" {
		t.Errorf("record = %s", body)
	}
	if got := configured(); !slices.Equal(got, []string{"thunderstoretrue", "nexustrue", "curseforgetrue"}) {
		t.Errorf("sites = %v", got)
	}
	edit.Nexus = NexusIDs{}
	edit.CurseForge = CurseForgeIDs{}
	if _, err := svc.Edit(t.Context(), "lethal-company", dir, edit); err != nil {
		t.Fatal(err)
	}
	body, _ = safefs.ReadFile(filepath.Join(dir, ".mortar", "template.json"))
	if strings.Contains(string(body), "nexus") || strings.Contains(string(body), "curseforge") {
		t.Errorf("cleared ids stay in %s", body)
	}
	edit.Nexus.FileID = "../x"
	if _, err := svc.Edit(t.Context(), "lethal-company", dir, edit); usererr.KindOf(err) != usererr.Invalid {
		t.Errorf("unsafe id: %v", err)
	}
}

func TestSiteIDsSurviveTheStartUpMigrationOfAProjectOfFiles(t *testing.T) {
	dir := t.TempDir()
	rec := `{"kind":"generic","name":"Pack","team":"Ab","nexus":{"modId":"5","fileId":"6"},"curseforge":{"projectId":7}}`
	if err := writeTracked(dir, assetRecordFile, []byte(rec)); err != nil {
		t.Fatal(err)
	}
	body, err := standardRecord(dir, "lethal-company")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeTracked(dir, assetRecordFile, body); err != nil {
		t.Fatal(err)
	}
	if got := readSites(dir); got.Nexus != (NexusIDs{ModID: "5", FileID: "6"}) || got.CurseForge.ProjectID != 7 {
		t.Errorf("ids after migration = %+v (%s)", got, body)
	}
}
