package control

import (
	"net"
	"strings"
	"testing"
	"time"
)

func readOver(t *testing.T, payload string, wait time.Duration) (request, error) {
	t.Helper()
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })
	go func() { _, _ = client.Write([]byte(payload)); _ = client.Close() }()
	return readRequest(server, server, "tok", wait)
}

func TestReadRequestBoundsAConnectionUntilItShowsTheToken(t *testing.T) {
	t.Parallel()
	req, err := readOver(t, `{"token":"tok","method":"m","params":{"name":"`+strings.Repeat("x", 1<<20)+`"}}`, time.Second)
	if err != nil || req.Method != "m" || len(req.Params.Name) != 1<<20 {
		t.Fatalf("a verified sender's large request: %v", err)
	}
	if _, err := readOver(t, `{"method":"m","pad":"`+strings.Repeat("x", 1<<20)+`"}`, time.Second); err == nil {
		t.Fatal("a large body without the token was read")
	}
	if _, err := readOver(t, `{"token":"bad","method":"m"}`, time.Second); err != errUnauthorized {
		t.Fatalf("wrong token: %v", err)
	}
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })
	if _, err := readRequest(server, server, "tok", 20*time.Millisecond); err == nil {
		t.Fatal("a silent connection was not cut off")
	}
}
