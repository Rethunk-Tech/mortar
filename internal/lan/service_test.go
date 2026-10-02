package lan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/share"
	"github.com/Rethunk-AI/mortar/internal/store"
)

func testPayload(t *testing.T) string {
	t.Helper()
	encoded, err := share.Encode(profile.Profile{Name: "Farm friends"})
	if err != nil {
		t.Fatal(err)
	}
	return encoded.Payload
}

func TestValidateRequest(t *testing.T) {
	payload := testPayload(t)
	shared, err := validateRequest(shareRequest{Sender: "Alex", Game: "stardew", Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if shared.Name != "Farm friends" {
		t.Fatalf("profile name = %q", shared.Name)
	}

	for _, request := range []shareRequest{
		{Sender: "Alex", Game: "stardew", Payload: "mortar://stardew/p/" + payload},
		{Sender: "Alex", Game: "other", Payload: payload},
		{Sender: "Alex", Game: "stardew", Payload: "not a payload"},
	} {
		if _, err := validateRequest(request); err == nil {
			t.Fatalf("validateRequest(%+v) accepted invalid input", request)
		}
	}
}

func TestValidateRequestCapsPayload(t *testing.T) {
	_, err := validateRequest(shareRequest{
		Sender:  "Alex",
		Game:    "stardew",
		Payload: strings.Repeat("A", maxPayloadBytes+1),
	})
	if !errors.Is(err, errPayloadTooLarge) {
		t.Fatalf("error = %v, want %v", err, errPayloadTooLarge)
	}
}

func TestAccountProof(t *testing.T) {
	proof := hmacProof("nexus-key", "nonce", "payload")
	if !accountMatches("nexus-key", "nonce", "payload", proof) {
		t.Fatal("matching proof was rejected")
	}
	if accountMatches("other-key", "nonce", "payload", proof) || accountMatches("", "nonce", "payload", proof) {
		t.Fatal("mismatched or missing key was accepted")
	}
	if accountMatches("nexus-key", "other-nonce", "payload", proof) || accountMatches("nexus-key", "nonce", "other", proof) {
		t.Fatal("mismatched message was accepted")
	}
}

func TestTransferTokenScope(t *testing.T) {
	service := NewService(Deps{})
	service.rememberGrant("token", "stardew", []string{"nexus-1-2"})

	request := httptest.NewRequestWithContext(context.Background(), "GET", "http://mortar.test/store/stardew/nexus-9-9", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	service.handleStore(response, request)
	if response.Code != 403 {
		t.Fatalf("unlisted key status = %d, want 403", response.Code)
	}

	request = httptest.NewRequestWithContext(context.Background(), "GET", "http://mortar.test/store/stardew/../nexus-1-2", nil)
	request.Header.Set("Authorization", "Bearer token")
	response = httptest.NewRecorder()
	service.handleStore(response, request)
	if response.Code != 400 {
		t.Fatalf("traversal status = %d, want 400", response.Code)
	}
}

func TestShareRateLimit(t *testing.T) {
	payload := testPayload(t)
	var arrivals []Arrival
	service := NewService(Deps{Emit: func(_ string, data any) {
		arrival, ok := data.(Arrival)
		if !ok {
			t.Errorf("event data = %T, want Arrival", data)
			return
		}
		arrivals = append(arrivals, arrival)
	}})
	body, err := json.Marshal(shareRequest{Sender: "Alex", Game: "stardew", Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		request := httptest.NewRequestWithContext(context.Background(), "POST", "http://mortar.test/share", bytes.NewReader(body))
		request.RemoteAddr = "192.0.2.10:4000"
		response := httptest.NewRecorder()
		service.handleShare(response, request)
		want := 200
		if i == 1 {
			want = 429
		}
		if response.Code != want {
			t.Fatalf("request %d status = %d, want %d", i+1, response.Code, want)
		}
	}
	if len(arrivals) != 1 {
		t.Fatalf("arrivals = %d, want 1", len(arrivals))
	}
}

func TestLoopbackTransfer(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	senderStore, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "mod.dll"), []byte("from sender"), 0o600); err != nil {
		t.Fatal(err)
	}
	key := store.NexusKey(7, 2)
	if err := senderStore.AddDir("stardew", key, source); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_DATA_HOME", t.TempDir())
	receiverStore, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	arrivals := make(chan Arrival, 1)
	receiver := NewService(Deps{
		Store:    receiverStore,
		NexusKey: func() (string, error) { return "same-account", nil },
		Emit: func(_ string, data any) {
			if arrival, ok := data.(Arrival); ok {
				arrivals <- arrival
			}
		},
	})
	sender := NewService(Deps{
		Store:    senderStore,
		NexusKey: func() (string, error) { return "same-account", nil },
	})
	senderServer := httptest.NewServer(sender.handler())
	defer senderServer.Close()
	sender.mu.Lock()
	sender.enabled = true
	sender.listener = senderServer.Listener
	sender.mu.Unlock()
	receiverServer := httptest.NewServer(receiver.handler())
	defer receiverServer.Close()

	encoded, err := share.Encode(profile.Profile{
		Name:    "Farm friends",
		Entries: []profile.Entry{{Key: "mod", Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 2}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.sendPayload(strings.TrimPrefix(receiverServer.URL, "http://"), "stardew", encoded.Payload); err != nil {
		t.Fatal(err)
	}
	var arrival Arrival
	select {
	case arrival = <-arrivals:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for share")
	}
	if !arrival.SameAccount {
		t.Fatal("same-account share did not receive a transfer grant")
	}
	if err := receiver.Transfer(arrival.ID); err != nil {
		t.Fatal(err)
	}
	installed, err := receiverStore.Path("stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := fsx.ReadFile(filepath.Join(installed, "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "from sender" {
		t.Fatalf("installed contents = %q", got)
	}
}

func TestLoopbackSendReceive(t *testing.T) {
	payload := testPayload(t)
	arrivals := make(chan Arrival, 1)
	service := NewService(Deps{Emit: func(_ string, data any) {
		arrival, ok := data.(Arrival)
		if !ok {
			t.Errorf("event data = %T, want Arrival", data)
			return
		}
		arrivals <- arrival
	}})
	server := httptest.NewServer(service.handler())
	defer server.Close()

	if err := service.sendPayload(strings.TrimPrefix(server.URL, "http://"), "stardew", payload); err != nil {
		t.Fatal(err)
	}
	select {
	case arrival := <-arrivals:
		if arrival.Sender != service.name || arrival.ProfileName != "Farm friends" {
			t.Fatalf("arrival = %+v", arrival)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for arrival")
	}
}
