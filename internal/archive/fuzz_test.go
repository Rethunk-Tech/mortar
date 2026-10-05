package archive

import (
	"errors"
	"strings"
	"testing"
)

func FuzzCleanName(f *testing.F) {
	for _, s := range []string{"a/b.txt", "../x", `..\x`, "/abs", "a/./b", "a//b", "a\x00b", "CON", "a/NUL.txt", ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) {
		got, err := cleanName(name)
		if err != nil {
			if !errors.Is(err, ErrTraversal) && !errors.Is(err, ErrUnsafeName) {
				t.Fatalf("untyped error for %q: %v", name, err)
			}
			return
		}
		if strings.HasPrefix(got, "/") || strings.Contains(got, `\`) || strings.ContainsRune(got, 0) {
			t.Fatalf("unsafe result %q from %q", got, name)
		}
		for s := range strings.SplitSeq(got, "/") {
			if s == ".." {
				t.Fatalf("escaping result %q from %q", got, name)
			}
		}
	})
}
