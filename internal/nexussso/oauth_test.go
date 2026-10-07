package nexussso

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

func TestOAuthPKCEAndRefresh(t *testing.T) {
	keyring.MockInit()
	var challenge string
	var forms []url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		forms = append(forms, r.PostForm)
		if r.PostForm.Get("grant_type") == "authorization_code" {
			sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
			if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"A1","refresh_token":"R1","expires_in":30}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"A2","expires_in":3600}`))
	}))
	defer srv.Close()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	o := OAuth{ClientID: "cid", AuthBase: srv.URL, Now: func() time.Time { return now }}
	o.OpenBrowser = func(raw string) error {
		u, _ := url.Parse(raw)
		q := u.Query()
		challenge = q.Get("code_challenge")
		if q.Get("redirect_uri") != RedirectURI || q.Get("scope") != "public openid profile" {
			t.Errorf("redirect %q scope %q", q.Get("redirect_uri"), q.Get("scope"))
		}
		if q.Get("client_id") != "cid" || q.Get("code_challenge_method") != "S256" {
			t.Errorf("authorize query %v", q)
		}
		go func() {
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, q.Get("redirect_uri")+"?code=CODE&state="+q.Get("state"), nil)
			if resp, err := http.DefaultClient.Do(req); err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
	tok, err := o.Authorize(context.Background())
	if err != nil || tok.Access != "A1" || tok.Refresh != "R1" || !tok.Expires.Equal(now.Add(30*time.Second)) {
		t.Fatalf("tokens %+v err %v", tok, err)
	}
	if err := Save(tok); err != nil {
		t.Fatal(err)
	}
	got, err := o.Fresh(context.Background())
	if err != nil || got.Access != "A2" || got.Refresh != "R1" {
		t.Fatalf("refreshed %+v err %v", got, err)
	}
	if forms[len(forms)-1].Get("refresh_token") != "R1" {
		t.Fatalf("refresh form %v", forms[len(forms)-1])
	}
	if stored, _ := Load(); stored.Access != "A2" {
		t.Fatalf("not re-saved: %+v", stored)
	}
}

func TestOAuthTimeoutAndDeniedCallback(t *testing.T) {
	o := OAuth{ClientID: "c", Timeout: 50 * time.Millisecond, OpenBrowser: func(string) error { return nil }}
	if _, err := o.Authorize(context.Background()); !errors.Is(err, ErrTimeout) {
		t.Fatalf("timeout: %v", err)
	}
	o.Timeout = time.Second
	o.OpenBrowser = func(raw string) error {
		u, _ := url.Parse(raw)
		go func() {
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, u.Query().Get("redirect_uri")+"?error=access_denied&state="+u.Query().Get("state"), nil)
			if resp, err := http.DefaultClient.Do(req); err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
	if _, err := o.Authorize(context.Background()); err == nil {
		t.Fatal("denied callback succeeded")
	}
}

func TestRedirectURIIsTheFixedConstant(t *testing.T) {
	if RedirectURI != "http://127.0.0.1:51762/oauth/callback" {
		t.Fatalf("redirect %q", RedirectURI)
	}
}

func TestBusyPortFailsWithoutFallingBack(t *testing.T) {
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", redirectAddr)
	if err != nil {
		t.Skipf("port %d is not free on this machine: %v", RedirectPort, err)
	}
	defer func() { _ = ln.Close() }()
	opened := false
	o := OAuth{ClientID: "c", OpenBrowser: func(string) error { opened = true; return nil }}
	_, err = o.Authorize(context.Background())
	if !errors.Is(err, ErrPortBusy) || err.Error() != "Port 51762 is in use; close the app using it and try again" {
		t.Fatalf("error %v", err)
	}
	if opened {
		t.Fatal("the browser opened although the callback could not listen")
	}
}
