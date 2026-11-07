package lan

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
)

// sealedStoreServer serves h the way a sender serves a store pull: sealed under a fresh transfer key, which it
// returns, for the token "t".
func sealedStoreServer(t *testing.T, h http.Handler) (*httptest.Server, []byte) {
	t.Helper()
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	sender := NewService(Deps{})
	sender.rememberGrant("t", "", nil, key)
	server := httptest.NewServer(sender.sealed(sender.resolveGrant, false, h.ServeHTTP))
	t.Cleanup(server.Close)
	return server, key
}
