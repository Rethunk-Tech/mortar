package github

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

const secret = "gho_testsecret123"

func TestDownloadErrorOmitsToken(t *testing.T) {
	old := DefaultAuth
	DefaultAuth = &Auth{Run: func(context.Context) ([]byte, error) { return []byte(secret), nil }}
	t.Cleanup(func() { DefaultAuth = old })
	hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") == "" {
			t.Error("token not sent")
		}
		return nil, errors.New("boom")
	})}
	err := Download(context.Background(), hc, "https://github.com/o/r/releases/download/v1/a.zip", t.TempDir()+"/a.zip", 1<<20, nil)
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("err %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
