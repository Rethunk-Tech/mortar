package lan

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

type outgoingRig struct {
	sender, receiver *Service
	arrival          Arrival
	events           func() []OutgoingTransfer
}

// sentRig pairs two computers and sends a profile of two Nexus files from the sender; the receiver already holds
// the first one.
func sentRig(t *testing.T) outgoingRig {
	t.Helper()
	senderStore, receiverStore := store.OpenAt(t.TempDir()), store.OpenAt(t.TempDir())
	var entries []profile.Entry
	for i, size := range []int{10, 30} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "m.dll"), bytes.Repeat([]byte("x"), size), 0o600); err != nil {
			t.Fatal(err)
		}
		key := store.NexusKey(7, i+1)
		if err := senderStore.AddDir("stardew", key, dir); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if err := receiverStore.AddDir("stardew", key, dir); err != nil {
				t.Fatal(err)
			}
		}
		entries = append(entries, profile.Entry{Key: key, Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: i + 1}})
	}
	var payload bytes.Buffer
	if _, err := share.Write(&payload, "stardew", profile.Profile{Name: "Farm friends", Entries: entries}, t.TempDir(), share.Include{}); err != nil {
		t.Fatal(err)
	}
	arrivals := make(chan Arrival, 1)
	var mu sync.Mutex
	var seen []OutgoingTransfer
	receiver, receiverAddr := pairedService(t, receiverStore, func(_ string, data any) {
		if a, ok := data.(Arrival); ok {
			arrivals <- a
		}
	})
	sender, _ := pairedService(t, senderStore, func(name string, data any) {
		if o, ok := data.(OutgoingTransfer); ok && name == OutgoingEvent {
			mu.Lock()
			seen = append(seen, o)
			mu.Unlock()
		}
	})
	pair(t, receiver, sender, receiverAddr)
	if err := sender.sendPayload(t.Context(), receiverAddr, "stardew", fixed(payload.Bytes())); err != nil {
		t.Fatal(err)
	}
	return outgoingRig{sender: sender, receiver: receiver, arrival: <-arrivals, events: func() []OutgoingTransfer {
		mu.Lock()
		defer mu.Unlock()
		return append([]OutgoingTransfer(nil), seen...)
	}}
}

func TestSenderSeesThePullFinish(t *testing.T) {
	r := sentRig(t)
	if got := r.sender.OutgoingTransfers(); len(got) != 1 || got[0].State != OutgoingSending || got[0].Total != 2 || got[0].Profile != "Farm friends" {
		t.Fatalf("right after sending = %+v", got)
	}
	if err := r.receiver.Transfer(t.Context(), r.arrival.ID); err != nil {
		t.Fatal(err)
	}
	got := r.sender.OutgoingTransfers()
	// The receiver held the first file, so only the second was pulled.
	want := OutgoingTransfer{ID: got[0].ID, Peer: got[0].Peer, Profile: "Farm friends", Current: 1, Total: 1, Bytes: 30, TotalBytes: 30, State: OutgoingDone}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("after the pull = %+v, want %+v", got, want)
	}
	events := r.events()
	if last := events[len(events)-1]; last != want {
		t.Fatalf("last event = %+v", last)
	}
}

func TestSenderSeesWhyThePullStopped(t *testing.T) {
	for _, tc := range []struct {
		name   string
		report outgoingReport
		want   OutgoingTransfer
	}{
		{"cancelled", outgoingReport{State: OutgoingCancelled}, OutgoingTransfer{State: OutgoingCancelled}},
		{"failed", outgoingReport{State: OutgoingFailed, Reason: "disk full"}, OutgoingTransfer{State: OutgoingFailed, Reason: "disk full"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := sentRig(t)
			incoming := r.receiver.incoming[r.arrival.ID]
			r.receiver.reportOutgoing(t.Context(), incoming, tc.report)
			got := r.sender.OutgoingTransfers()[0]
			if got.State != tc.want.State || got.Reason != tc.want.Reason {
				t.Fatalf("transfer = %+v", got)
			}
			// A finished transfer keeps its ending.
			r.receiver.reportOutgoing(t.Context(), incoming, outgoingReport{State: OutgoingDone})
			if again := r.sender.OutgoingTransfers()[0]; again.State != tc.want.State {
				t.Fatalf("a late report changed it to %+v", again)
			}
		})
	}
}

func TestSenderGivesUpOnAPullThatWentQuiet(t *testing.T) {
	r := sentRig(t)
	for token := range r.sender.outgoing {
		r.sender.outgoing[token].stall.Reset(time.Millisecond)
	}
	deadline := time.Now().Add(2 * time.Second)
	for r.sender.OutgoingTransfers()[0].State == OutgoingSending {
		if time.Now().After(deadline) {
			t.Fatal("the transfer never gave up")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := r.sender.OutgoingTransfers()[0]; got.State != OutgoingFailed || got.Reason == "" {
		t.Fatalf("transfer = %+v", got)
	}
}

func TestSenderGoingAwayMidTransferSaysSo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Promising more than is sent makes the server drop the connection, as a sender that shut down would.
		w.Header().Set("Content-Length", "100000")
		_, _ = w.Write([]byte("partial"))
	}))
	defer server.Close()
	receiver := NewService(Deps{Store: store.OpenAt(t.TempDir())})
	incoming := incomingTransfer{
		Peer: strings.TrimPrefix(server.URL, "http://"), Sender: "Desk", Game: "stardew", Token: "t",
		Items: []transferItem{{Key: store.NexusKey(7, 1), Hash: "h"}},
	}
	_, err := receiver.fetchEntry(t.Context(), incoming, incoming.Items[0], 1, 0, 1, 0, time.Now())
	if usererr.KindOf(err) != usererr.Network || !strings.Contains(err.Error(), "Desk went away") {
		t.Fatalf("mid-transfer disconnect = %v (kind %s)", err, usererr.KindOf(err))
	}
}

func TestADamagedFileIsNotCalledADisconnect(t *testing.T) {
	err := receiveError("Desk", errors.New("tar: invalid header"))
	if usererr.KindOf(err) != usererr.Damaged || !strings.Contains(err.Error(), "Desk") {
		t.Fatalf("error = %v (kind %s)", err, usererr.KindOf(err))
	}
	if kind := usererr.KindOf(receiveError("Desk", syscall.ENOSPC)); kind != usererr.DiskFull {
		t.Fatalf("disk full classified as %s", kind)
	}
}
