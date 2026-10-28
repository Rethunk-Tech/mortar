package components

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/meta"
)

func TestDecodeLogSuppressionsRefusesAnotherGameAndAnUnlabelledEntry(t *testing.T) {
	for name, body := range map[string]string{
		"another game": `{"game":"valheim","serial":1,"entries":[]}`,
		"no serial":    `{"game":"lc","entries":[]}`,
		"no label":     `{"game":"lc","serial":1,"entries":[{"source":"T::Open"}]}`,
	} {
		if _, err := DecodeLogSuppressions([]byte(body), "lc"); err == nil {
			t.Errorf("%s: decoded", name)
		}
	}
	l, err := DecodeLogSuppressions([]byte(`{"game":"lc","serial":2,"entries":[{"source":"T::Open","label":"Door"}]}`), "lc")
	if err != nil || len(l.Entries) != 1 {
		t.Fatalf("list = %+v, %v", l, err)
	}
}

// A caller whose context is already cancelled does not leave an empty list in the memo for the next hour.
func TestListMemoDoesNotKeepAListFromACancelledCaller(t *testing.T) {
	UseSignedLists(NewClient(nil), &meta.Client{CacheDir: t.TempDir()}, nil)
	t.Cleanup(func() { UseSignedLists(nil, nil, nil) })
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	LogSuppressionsFor(ctx, "lc")
	suppressionLists.mu.Lock()
	defer suppressionLists.mu.Unlock()
	if _, ok := suppressionLists.lists["lc"]; ok {
		t.Fatal("a cancelled fetch was memoised")
	}
}
