package tsupload

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestRefusedTokenReadsTheStatus(t *testing.T) {
	for code, want := range map[int]bool{http.StatusUnauthorized: true, http.StatusForbidden: true, http.StatusBadGateway: false} {
		err := refusedToken(fmt.Errorf("upload: %w", &httpkit.StatusError{Code: code, Status: http.StatusText(code)}))
		if got := usererr.KindOf(err) == usererr.Permission; got != want {
			t.Errorf("%d: permission = %v, want %v", code, got, want)
		}
	}
}
