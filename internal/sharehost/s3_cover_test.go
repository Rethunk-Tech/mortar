package sharehost

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestS3ObjectKeys(t *testing.T) {
	if got := metaKey("abc"); got != "shares/abc/meta.json" {
		t.Fatalf("metaKey=%q", got)
	}
	if got := verKey("abc", 3); got != "shares/abc/3.json" {
		t.Fatalf("verKey=%q", got)
	}
}

func TestShareCodeFromObjectKey(t *testing.T) {
	tests := []struct {
		key  string
		code string
		ok   bool
	}{
		{"shares/abc/meta.json", "abc", true},
		{"shares/meta.json", "meta.json", true},
		{"shares//meta.json", "", false},
		{"shares/a/b/meta.json", "", false},
		{"shares/abc/1.json", "", false},
		{"other/abc/meta.json", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		code, ok := shareCodeFromObjectKey(tt.key)
		if code != tt.code || ok != tt.ok {
			t.Fatalf("%q: got %q %v", tt.key, code, ok)
		}
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{minio.ErrorResponse{StatusCode: http.StatusNotFound}, true},
		{minio.ErrorResponse{Code: "NoSuchKey"}, true},
		{minio.ErrorResponse{Code: "NoSuchObject"}, true},
		{minio.ErrorResponse{StatusCode: http.StatusInternalServerError, Code: "InternalError"}, false},
		{errors.New("nope"), false},
		{nil, false},
	}
	for _, tt := range tests {
		if got := isNotFound(tt.err); got != tt.want {
			t.Fatalf("%v: got %v", tt.err, got)
		}
	}
}

func TestNewS3Store(t *testing.T) {
	s, err := NewS3Store("https://nyc3.digitaloceanspaces.com", "bucket", "ak", "sk")
	if err != nil {
		t.Fatal(err)
	}
	if s.bucket != "bucket" || s.client == nil {
		t.Fatalf("store=%+v", s)
	}
	if _, err := NewS3Store("http://example.com", "b", "ak", "sk"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewS3Store("", "b", "ak", "sk"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPutJSONMarshalError(t *testing.T) {
	s := &S3Store{}
	if err := s.putJSON(t.Context(), "k", make(chan int)); err == nil {
		t.Fatal("expected error")
	}
}

const s3LastMod = "Wed, 01 Jan 2020 00:00:00 GMT"

type memS3 struct {
	mu      sync.Mutex
	bucket  string
	objects map[string][]byte
	failGet string
	failPut bool
	failDel bool
	failLst error
}

func (m *memS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/"+m.bucket)
	path = strings.TrimPrefix(path, "/")
	if r.Method == http.MethodGet && (r.URL.Query().Get("list-type") != "" || r.URL.Path == "/"+m.bucket || r.URL.Path == "/"+m.bucket+"/") {
		m.list(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		if m.failPut {
			http.Error(w, "put failed", http.StatusInternalServerError)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if i := bytes.IndexByte(body, '{'); i >= 0 {
			if j := bytes.LastIndexByte(body, '}'); j > i {
				body = body[i : j+1]
			}
		}
		m.mu.Lock()
		m.objects[path] = body
		m.mu.Unlock()
		w.Header().Set("ETag", `"etag"`)
		w.Header().Set("Last-Modified", s3LastMod)
		w.WriteHeader(http.StatusOK)
	case http.MethodGet, http.MethodHead:
		if m.failGet == path || m.failGet == "*" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`<Error><Code>InternalError</Code><Message>boom</Message></Error>`))
			return
		}
		m.mu.Lock()
		body, ok := m.objects[path]
		m.mu.Unlock()
		if !ok {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`<Error><Code>NoSuchKey</Code><Message>not found</Message></Error>`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"etag"`)
		w.Header().Set("Last-Modified", s3LastMod)
		w.Header().Set("Content-Length", itoa(len(body)))
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write(body)
	case http.MethodDelete:
		if m.failDel {
			http.Error(w, "del failed", http.StatusInternalServerError)
			return
		}
		m.mu.Lock()
		delete(m.objects, path)
		m.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *memS3) list(w http.ResponseWriter, r *http.Request) {
	if m.failLst != nil {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<Error><Code>InternalError</Code><Message>` + m.failLst.Error() + `</Message></Error>`))
		return
	}
	prefix := r.URL.Query().Get("prefix")
	type content struct {
		XMLName      xml.Name `xml:"Contents"`
		Key          string   `xml:"Key"`
		LastModified string   `xml:"LastModified"`
		ETag         string   `xml:"ETag"`
		Size         int      `xml:"Size"`
		StorageClass string   `xml:"StorageClass"`
	}
	var contents []content
	m.mu.Lock()
	for k, v := range m.objects {
		if strings.HasPrefix(k, prefix) {
			contents = append(contents, content{
				Key: k, LastModified: "2020-01-01T00:00:00.000Z", ETag: `"etag"`, Size: len(v), StorageClass: "STANDARD",
			})
		}
	}
	m.mu.Unlock()
	type result struct {
		XMLName     xml.Name  `xml:"ListBucketResult"`
		Name        string    `xml:"Name"`
		Prefix      string    `xml:"Prefix"`
		KeyCount    int       `xml:"KeyCount"`
		MaxKeys     int       `xml:"MaxKeys"`
		IsTruncated bool      `xml:"IsTruncated"`
		Contents    []content `xml:",omitempty"`
	}
	w.Header().Set("Content-Type", "application/xml")
	_ = xml.NewEncoder(w).Encode(result{
		Name: m.bucket, Prefix: prefix, KeyCount: len(contents), MaxKeys: 1000, Contents: contents,
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func testS3(t *testing.T, mem *memS3) *S3Store {
	t.Helper()
	if mem.objects == nil {
		mem.objects = map[string][]byte{}
	}
	if mem.bucket == "" {
		mem.bucket = "bucket"
	}
	srv := httptest.NewServer(mem)
	t.Cleanup(srv.Close)
	minio.MaxRetry = 1
	client, err := minio.New(strings.TrimPrefix(srv.URL, "http://"), &minio.Options{
		Creds:  credentials.NewStaticV4("ak", "sk", ""),
		Secure: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &S3Store{client: client, bucket: mem.bucket}
}

func TestS3StoreRoundTripAndSweep(t *testing.T) {
	mem := &memS3{}
	s := testS3(t, mem)
	ctx := t.Context()
	old := Meta{TokenHash: "t", Latest: 1, Created: time.Now().Add(-48 * time.Hour)}
	fresh := Meta{TokenHash: "t", Latest: 1, Created: time.Now(), LastRead: time.Now()}
	ver := Version{Payload: "p", Game: "stardew"}

	if _, err := s.Meta(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Meta missing: %v", err)
	}
	if err := s.Create(ctx, "old", old, ver); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(ctx, "old", old, ver); !errors.Is(err, ErrExists) {
		t.Fatalf("Create exists: %v", err)
	}
	if err := s.Create(ctx, "fresh", fresh, ver); err != nil {
		t.Fatal(err)
	}
	got, err := s.Meta(ctx, "old")
	if err != nil || got.Latest != 1 {
		t.Fatalf("Meta: %+v, %v", got, err)
	}
	if err := s.SaveMeta(ctx, "old", old); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveVersion(ctx, "old", 2, ver); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.LoadVersion(ctx, "old", 1)
	if err != nil || loaded.Game != "stardew" {
		t.Fatalf("LoadVersion: %+v, %v", loaded, err)
	}
	if err := s.SaveMeta(ctx, "missing", old); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SaveMeta missing: %v", err)
	}
	if err := s.SaveVersion(ctx, "missing", 1, ver); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SaveVersion missing: %v", err)
	}
	if _, err := s.LoadVersion(ctx, "missing", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadVersion missing: %v", err)
	}

	mem.mu.Lock()
	mem.objects["shares/nested/x/meta.json"] = []byte(`{}`)
	mem.objects["shares/skip.json"] = []byte(`{}`)
	mem.mu.Unlock()
	deleted, kept, err := s.Sweep(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 || kept != 1 {
		t.Fatalf("Sweep = %d deleted, %d kept", deleted, kept)
	}
}

func TestS3GetAndListErrors(t *testing.T) {
	mem := &memS3{failGet: "*"}
	s := testS3(t, mem)
	if _, err := s.Meta(t.Context(), "x"); err == nil {
		t.Fatal("expected get error")
	}
	mem.failGet = ""
	mem.failLst = errors.New("list")
	if _, _, err := s.Sweep(t.Context(), time.Now()); err == nil {
		t.Fatal("expected list error")
	}
	mem.failLst = nil
	mem.objects["shares/bad/meta.json"] = []byte(`{`)
	if _, _, err := s.Sweep(t.Context(), time.Now()); err == nil {
		t.Fatal("expected invalid meta")
	}
}
