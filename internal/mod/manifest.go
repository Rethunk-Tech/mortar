package mod

import (
	"bytes"
	"encoding/json"
)

// BOM is the UTF-8 byte order mark many mods' XML, JSON and ini files start with.
const BOM = "\xef\xbb\xbf"

// StripBOM returns b without a leading BOM.
func StripBOM(b []byte) []byte { return bytes.TrimPrefix(b, []byte(BOM)) }

// UnmarshalManifest decodes a Thunderstore manifest.json into v. Thunderstore accepts manifests saved with a UTF-8
// BOM, and many packages ship one, but encoding/json refuses it.
func UnmarshalManifest(b []byte, v any) error {
	return json.Unmarshal(StripBOM(b), v)
}
