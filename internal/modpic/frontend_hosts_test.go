package modpic

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
)

func TestFrontendAllowedHostsMatch(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "frontend", "src", "browse", "pictureSrc.ts"))
	if err != nil {
		t.Fatal(err)
	}
	start := bytes.Index(src, []byte("new Set(["))
	if start < 0 {
		t.Fatal("pictureSrc.ts: allowedHosts Set not found")
	}
	end := bytes.Index(src[start:], []byte("])"))
	if end < 0 {
		t.Fatal("pictureSrc.ts: allowedHosts Set not closed")
	}
	matches := regexp.MustCompile(`'([^']+)'`).FindAllSubmatch(src[start:start+end], -1)
	front := make([]string, 0, len(matches))
	for _, m := range matches {
		front = append(front, string(m[1]))
	}
	if !slices.Equal(front, allowedHosts) {
		t.Fatalf("frontend allowedHosts = %q, want %q", front, allowedHosts)
	}
}
