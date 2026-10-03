package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCappedWriterStopsAtLimitWithOneNotice(t *testing.T) {
	var buf bytes.Buffer
	c := &cappedWriter{w: &buf, left: 10}
	for _, s := range []string{"12345", "6789", "abcdef", "more"} {
		if n, err := c.Write([]byte(s)); err != nil || n != len(s) {
			t.Fatalf("write %q: %d %v", s, n, err)
		}
	}
	got := buf.String()
	if !strings.HasPrefix(got, "123456789") || strings.Count(got, "size limit") != 1 || strings.Contains(got, "abcdef") {
		t.Fatalf("got %q", got)
	}
}
