package sampler

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func FuzzParse(f *testing.F) {
	trace, err := os.ReadFile("testdata/spin.nettrace")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(trace)
	f.Add(trace[:len(trace)/2])
	f.Add(trace[:200])
	f.Add([]byte("Nettrace"))
	mangled := bytes.Clone(trace[:16<<10])
	copy(mangled[12:32], strings.Repeat("x", 20))
	f.Add(mangled)
	zeroed := bytes.Clone(trace[:16<<10])
	clear(zeroed[8:64])
	f.Add(zeroed)
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(bytes.NewReader(data))
	})
}

func TestParseRefusesStreamsWithoutBlocks(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		data []byte
		want string
	}{
		"not nettrace":     {[]byte("MZ not a trace at all"), "not a nettrace stream"},
		"too short":        {[]byte("Nettra"), "not a nettrace stream"},
		"non-V5 header":    {append([]byte("Nettrace"), bytes.Repeat([]byte{0}, 64)...), "no event or stack blocks"},
		"wrong serializer": {append([]byte("Nettrace\x00\x00\x00\x00!FastSerialization.9"), bytes.Repeat([]byte{0}, 64)...), "no event or stack blocks"},
	} {
		if _, err := Parse(bytes.NewReader(tc.data)); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tc.want)
		}
	}
}
