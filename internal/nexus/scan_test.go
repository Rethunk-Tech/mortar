package nexus

import (
	"context"
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
