package mod

import (
	"bytes"
	"encoding/json"
)

// UnmarshalManifest decodes a Thunderstore manifest.json into v. Thunderstore accepts manifests saved with a UTF-8
// BOM, and many packages ship one, but encoding/json refuses it.
func UnmarshalManifest(b []byte, v any) error {
	return json.Unmarshal(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")), v)
}
