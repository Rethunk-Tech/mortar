package lan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/share"
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
		want := 204
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
