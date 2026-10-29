package thunderstore

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestAnOversizeAnswerIsRefusedNotCutShort(t *testing.T) {
	t.Parallel()
	body := make([]byte, source.MaxBody+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
	defer srv.Close()
	if _, err := (Driver{}).get(context.Background(), srv.URL, "ua"); !errors.Is(err, safefs.ErrTooLarge) {
		t.Fatalf("got %v", err)
	}
}
