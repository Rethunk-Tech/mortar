package bytefmt

import "testing"

func TestFormat(t *testing.T) {
	for n, want := range map[int64]string{12: "12 B", 1024: "1.0 KB", 70 << 20: "70.0 MB", 3 << 30: "3.0 GB"} {
		if got := Format(n); got != want {
			t.Errorf("Format(%d) = %q, want %q", n, got, want)
		}
	}
}
