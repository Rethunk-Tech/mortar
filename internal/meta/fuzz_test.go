package meta

import (
	"net/url"
	"testing"
)

// FuzzDecodeCollection feeds Nexus's answer for a collection revision: its mod list, curator instructions and the
// resources to install by hand.
func FuzzDecodeCollection(f *testing.F) {
	f.Add([]byte(`{"data":{"collectionRevision":{"revisionNumber":4,"downloadLink":"/v2/collections/1/revisions/4/download_link","installationInfo":" Read me ",` +
		`"externalResources":[{"name":"Tool","resourceType":"ExternalSite","resourceUrl":"https://example.com/t","optional":true},{"name":"Bad","resourceUrl":"ms-settings:"}],` +
		`"collection":{"name":"Cozy Farm","slug":"cozy-farm","user":{"name":"Pat"}},"modFiles":[{"fileId":11,"optional":false,"file":{"modId":100}},{"fileId":33,"file":null}]}}}`))
	f.Add([]byte(`{"errors":[{"message":"not found"}]}`))
	f.Add([]byte(`{"data":{"collectionRevision":null}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		c, err := decodeCollection("slug", raw)
		if err != nil {
			return
		}
		for _, e := range c.External {
			if e.URL == "" {
				continue
			}
			if u, err := url.Parse(e.URL); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
				t.Fatalf("kept resource link %q", e.URL)
			}
		}
	})
}
