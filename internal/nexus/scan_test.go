package nexus

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScanStatusesCachesGraphQLResponse(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost || r.URL.Path != "/v2/graphql" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"modFiles":[{"fileId":12,"scannedV2":"QUARANTINED"}]}}`))
	}))
	defer server.Close()

	client := New("test")
	client.BaseURL = server.URL
	first, err := client.ScanStatuses(context.Background(), stardew, 30597)
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.ScanStatuses(context.Background(), stardew, 30597)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 || first[12] != "QUARANTINED" || second[12] != "QUARANTINED" {
		t.Fatalf("requests=%d first=%v second=%v", requests, first, second)
	}
}

var stardew = Title{Domain: "stardewvalley", ID: 1303}

func TestScanStatusesTellARefusedKeyFromASpentQuota(t *testing.T) {
	code := http.StatusUnauthorized
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }))
	defer server.Close()
	client := New("test")
	client.BaseURL = server.URL
	if _, err := client.ScanStatuses(context.Background(), stardew, 1); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("a refused key: %v", err)
	}
	code = http.StatusTooManyRequests
	var limited *RateLimitError
	if _, err := client.ScanStatuses(context.Background(), stardew, 2); !errors.As(err, &limited) {
		t.Fatalf("a spent quota: %v", err)
	}
}
