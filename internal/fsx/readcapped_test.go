package fsx

import (
	"errors"
	"strings"
	"testing"
)

func TestReadCappedTakesInputUpToTheLimitAndRefusesOneByteMore(t *testing.T) {
	t.Parallel()
	if b, err := ReadCapped(strings.NewReader("abcd"), 4); err != nil || string(b) != "abcd" {
		t.Fatalf("at the limit: %q, %v", b, err)
	}
	if b, err := ReadCapped(strings.NewReader("abcde"), 4); !errors.Is(err, ErrTooLarge) || b != nil {
		t.Fatalf("one byte over: %q, %v", b, err)
	}
	if b, err := ReadCapped(strings.NewReader(""), 0); err != nil || len(b) != 0 {
		t.Fatalf("empty at a zero limit: %q, %v", b, err)
	}
}
