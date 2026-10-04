package meta

import (
	"os"
	"strings"
	"testing"
)

func TestDecodeCollection(t *testing.T) {
	body := []byte(`{"data":{"collectionRevision":{"revisionNumber":4,"collection":{"name":"Cozy Farm","slug":"cozy-farm","user":{"name":"Pat"}},"modFiles":[{"fileId":11,"optional":false,"file":{"modId":100,"name":"A","version":"1.0","mod":{"name":"A"}}},{"fileId":22,"optional":true,"file":{"modId":200}},{"fileId":33,"optional":false,"file":null}]}}}`)
	got, err := decodeCollection("cozy-farm", body)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Cozy Farm" || got.Author != "Pat" || got.Slug != "cozy-farm" || got.Revision != 4 {
		t.Fatalf("collection = %+v", got)
	}
	if len(got.Files) != 2 || got.Files[0] != (CollectionFile{ModID: 100, FileID: 11}) || got.Files[1] != (CollectionFile{ModID: 200, FileID: 22, Optional: true}) {
		t.Fatalf("files = %+v", got.Files)
	}

	_, err = decodeCollection("missing", []byte(`{"data":{"collectionRevision":null}}`))
	if err == nil || !strings.Contains(err.Error(), `nexus has no collection "missing"`) {
		t.Fatalf("null revision: %v", err)
	}
	_, err = decodeCollection("gone", []byte(`{"errors":[{"message":"not found"}]}`))
	if err == nil || !strings.Contains(err.Error(), `nexus has no collection "gone"`) {
		t.Fatalf("errors: %v", err)
	}
}

func TestDecodeCollectionRevisionFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/collection-revision.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeCollection("htknoa", raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Instructions != "Start a **new save**. See https://example.com/guide" || got.DownloadLink != "/v2/collections/636/revisions/948/download_link" {
		t.Fatalf("collection = %+v", got)
	}
	want := CollectionExternal{Name: "Hand Mod", Type: "browse", URL: "https://example.com/mod", Version: "1.2", Author: "Sam", Optional: true}
	if len(got.External) != 1 || got.External[0] != want || len(got.Files) != 3 {
		t.Fatalf("external = %+v files = %d", got.External, len(got.Files))
	}
}
