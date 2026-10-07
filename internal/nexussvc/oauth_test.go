package nexussvc

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/nexussso"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/zalando/go-keyring"
)

func TestOAuthBuildSendsBearerAndNeverTheKey(t *testing.T) {
	keyring.MockInit()
	old := nexussso.ClientID
	nexussso.ClientID = "cid"
	t.Cleanup(func() { nexussso.ClientID = old })
	if err := nexussso.Save(nexussso.Tokens{Access: "TOK", Refresh: "R", Expires: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = w.Write([]byte(`{"user_id":1,"name":"n","is_premium":false}`))
	}))
	defer srv.Close()

	store, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(func(v *settings.Settings) { v.NexusUserID = 1 }); err != nil {
		t.Fatal(err)
	}
	base := nexus.New("1")
	base.BaseURL, base.CacheDir = srv.URL, t.TempDir()
	c, err := Authed(store, base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Validate(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got.Get("Authorization") != "Bearer TOK" || got.Get("Apikey") != "" {
		t.Fatalf("headers %v", got)
	}
	if _, err := (&Service{}).SignIn(t.Context(), "pasted"); err == nil {
		t.Fatal("a pasted key signed in while OAuth is configured")
	}
}
