package ids

import (
	"regexp"
	"testing"
)

func TestNew(t *testing.T) {
	a, b := New(), New()
	if !regexp.MustCompile(`^[0-9a-f]{16}$`).MatchString(a) || a == b {
		t.Fatalf("New() = %q, %q", a, b)
	}
}

func TestIs(t *testing.T) {
	if !Is(New()) || Is("0123456789ABCDEF") || Is("short") || Is("0123456789abcdeg") {
		t.Fatal("Is does not match New's shape")
	}
}
