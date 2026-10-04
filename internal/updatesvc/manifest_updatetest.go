//go:build updatetest

package updatesvc

import "os"

// The updatetest tag exists only for the end-to-end updater test (e2e_test.go); release builds never set it, so the
// manifest URL cannot be redirected and server builds stay update-free there.
const serverUpdates = true

func manifestURL() string {
	if u := os.Getenv("MORTAR_UPDATE_MANIFEST_URL"); u != "" {
		return u
	}
	return ManifestURL
}
