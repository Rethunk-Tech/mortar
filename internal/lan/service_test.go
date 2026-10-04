package lan

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/hashicorp/mdns"
)

func testPayload(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	_, err := share.Write(&buf, profile.Profile{Name: "Farm friends"}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawStdEncoding.EncodeToString(buf.Bytes())
}

func TestValidateRequest(t *testing.T) {
	payload := testPayload(t)
	shared, err := validateRequest(shareRequest{Sender: "Alex", Game: "stardew", Payload: payload, Version: protocolVersion})
	if err != nil {
		t.Fatal(err)
	}
	if shared.Name != "Farm friends" {
		t.Fatalf("profile name = %q", shared.Name)
	}

	for _, request := range []shareRequest{
		{Sender: "Alex", Game: "stardew", Payload: "not-base64"},
		{Sender: "Alex", Game: "other", Payload: payload},
		{Sender: "Alex", Game: "stardew", Payload: payload, Version: "1"},
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
		Payload: base64.RawStdEncoding.EncodeToString(make([]byte, maxPayloadBytes+1)),
		Version: protocolVersion,
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
	body, err := json.Marshal(shareRequest{Sender: "Alex", Game: "stardew", Payload: payload, Version: protocolVersion})
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
	// An optional file laid over the main file travels as its own store item.
	optional := t.TempDir()
	if err := os.WriteFile(filepath.Join(optional, "alt.png"), []byte("optional"), 0o600); err != nil {
		t.Fatal(err)
	}
	optKey := store.NexusKey(7, 3)
	if err := senderStore.AddDir("stardew", optKey, optional); err != nil {
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

	var payload bytes.Buffer
	if _, err := share.Write(&payload, profile.Profile{
		Name: "Farm friends",
		Entries: []profile.Entry{
			{Key: key, Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 2}},
			{Key: optKey, Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 3}, OverlayOf: key, OverlayFrom: "a", OverlayTo: "b", OverlayOff: true},
		},
	}, t.TempDir(), share.Include{DisabledMods: true}); err != nil {
		t.Fatal(err)
	}
	if pv, err := share.ReadBytes(payload.Bytes()); err != nil || pv.Entries[1].Overlay == nil || *pv.Entries[1].Overlay != (share.Overlay{From: "a", To: "b", Off: true}) {
		t.Fatalf("payload overlay = %+v, %v", pv.Entries, err)
	}
	if err := sender.sendPayload(strings.TrimPrefix(receiverServer.URL, "http://"), "stardew", payload.Bytes()); err != nil {
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
	dir, err := receiverStore.Path("stardew", optKey)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := fsx.ReadFile(filepath.Join(dir, "alt.png")); err != nil || string(got) != "optional" {
		t.Fatalf("optional file did not transfer: %q %v", got, err)
	}
}

func TestLoopbackSendReceive(t *testing.T) {
	payload, err := base64.RawStdEncoding.DecodeString(testPayload(t))
	if err != nil {
		t.Fatal(err)
	}
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

func TestLoopbackLargeMortarRoundTrip(t *testing.T) {
	modsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(modsDir, "mod-000"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modsDir, "mod-000", "config.json"), []byte(`{"enabled":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	source := profile.Source{Kind: profile.KindNexus, ModID: 1, FileID: 1}
	entries := make([]profile.Entry, 600)
	for i := range entries {
		entries[i] = profile.Entry{
			Key:    fmt.Sprintf("mod-%03d", i),
			Source: source,
			Mods:   []profile.EntryMod{{UniqueID: fmt.Sprintf("mod.%03d", i), Folder: "."}},
		}
	}
	original := profile.Profile{
		Name:        "Large farm",
		Notes:       "Keep this note.",
		Description: "Profile settings survive LAN transfer.",
		Entries:     entries,
	}
	var archive bytes.Buffer
	if _, err := share.Write(&archive, original, modsDir); err != nil {
		t.Fatal(err)
	}
	arrivals := make(chan Arrival, 1)
	service := NewService(Deps{Emit: func(_ string, data any) {
		arrival, ok := data.(Arrival)
		if !ok {
			t.Errorf("event data type = %T, want Arrival", data)
			return
		}
		arrivals <- arrival
	}})
	server := httptest.NewServer(service.handler())
	defer server.Close()
	if err := service.sendPayload(strings.TrimPrefix(server.URL, "http://"), "stardew", archive.Bytes()); err != nil {
		t.Fatal(err)
	}
	arrival := <-arrivals
	raw, err := base64.RawStdEncoding.DecodeString(arrival.Payload)
	if err != nil {
		t.Fatal(err)
	}
	received, err := share.ReadBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	if received.Name != original.Name || received.Notes != original.Notes || received.Description != original.Description {
		t.Fatalf("profile metadata = %#v, want %#v", received, original)
	}
	if len(received.Entries) != len(original.Entries) || len(received.Configs) != 1 {
		t.Fatalf("received %d entries and %d configs, want %d entries and 1 config", len(received.Entries), len(received.Configs), len(original.Entries))
	}
}

func TestPeerNameUnescapesDNSInstanceName(t *testing.T) {
	if got := peerName(`Pat\ Farmer._mortar._tcp.local.`); got != "Pat Farmer" {
		t.Fatalf("peerName() = %q, want %q", got, "Pat Farmer")
	}
}

func TestPeerNameRejectsOtherServiceTypes(t *testing.T) {
	if got := peerName("OpenThread._meshcop._udp.local."); got != "" {
		t.Fatalf("peerName() = %q, want empty", got)
	}
}

func TestAddPeerDropsOurInstance(t *testing.T) {
	service := NewService(Deps{})
	service.enabled = true
	service.addPeer(&mdns.ServiceEntry{
		Name:       `Pat\ Farmer._mortar._tcp.local.`,
		Port:       1234,
		AddrV4:     net.ParseIP("192.0.2.1"),
		InfoFields: []string{"instance=" + service.instanceID},
	})
	if peers := service.Peers(); len(peers) != 0 {
		t.Fatalf("Peers() = %#v, want no peers", peers)
	}
}
