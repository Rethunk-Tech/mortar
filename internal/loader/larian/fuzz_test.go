package larian

import (
	"bytes"
	"testing"
)

// FuzzPak holds that no downloaded pak or meta.lsx makes the reader panic or allocate past its bounds.
func FuzzPak(f *testing.F) {
	f.Add([]byte("LSPK\x12\x00\x00\x00\x28\x00\x00\x00\x00\x00\x00\x00"), []byte(meta))
	f.Add([]byte("LSPK"), []byte(`<save><region id="Config"><node id="ModuleInfo"/></region></save>`))
	f.Fuzz(func(_ *testing.T, pak, lsx []byte) {
		if entries, err := pakEntries(bytes.NewReader(pak)); err == nil {
			for _, e := range entries {
				_, _ = decompress(make([]byte, min(e.size, maxMeta)), e.flags, int(min(e.uncompress, maxMeta)))
			}
		}
		_, _, _ = parseMeta(lsx)
	})
}
