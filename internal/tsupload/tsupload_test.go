package tsupload

import (
	"context"
	"net/http/httptest"

	"fmt"
	"github.com/zalando/go-keyring"
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

func TestUploadNeedsAToken(t *testing.T) {
	keyring.MockInit()
	_, err := Upload(context.Background(), Client, Site, "x.zip", "team", "c", nil)
	if usererr.KindOf(err) != usererr.Permission {
		t.Fatalf("err = %v", err)
	}
}

func TestSetTokenNeedsBoth(t *testing.T) {
	if usererr.KindOf(SetToken("", "t")) != usererr.Invalid || usererr.KindOf(SetToken("t", "")) != usererr.Invalid {
		t.Fatal("empty team or token accepted")
	}
}

func TestClientKeepsTheTokenOnItsHost(t *testing.T) {
	keyring.MockInit()
	if err := SetToken("team", "secret"); err != nil {
		t.Fatal(err)
	}
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("redirect followed with Authorization %q", r.Header.Get("Authorization"))
	}))
	defer other.Close()
	home := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusFound)
	}))
	defer home.Close()
	if _, err := Upload(context.Background(), Client, home.URL+"/", "x.zip", "team", "c", nil); err == nil {
		t.Fatal("cross-host redirect accepted")
	}
}
