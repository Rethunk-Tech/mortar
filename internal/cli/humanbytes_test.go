package cli

import "testing"

func TestHumanBytes(t *testing.T) {
	if got := humanBytes(12); got != "12 B" {
		t.Fatalf("12 -> %q", got)
	}
	if got := humanBytes(1024); got != "1.0 KB" {
		t.Fatalf("1024 -> %q", got)
	}
}
