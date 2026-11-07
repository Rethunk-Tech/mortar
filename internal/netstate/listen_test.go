package netstate

import "testing"

func TestPprofListensOnLoopbackUnlessTheHostIsNamed(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"6060":           "127.0.0.1:6060",
		":6060":          "127.0.0.1:6060",
		"0.0.0.0:6060":   "0.0.0.0:6060",
		"192.0.2.7:6060": "192.0.2.7:6060",
	} {
		if got := LoopbackUnlessNamed(in); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}
