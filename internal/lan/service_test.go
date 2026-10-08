package lan

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/hashicorp/mdns"
)

func TestMain(m *testing.M) {
	pairIterations = 1_000
	retryDelay = time.Millisecond
	os.Exit(m.Run())
}

func testPayload(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	_, err := share.Write(&buf, "stardew", profile.Profile{Name: "Farm friends"}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawStdEncoding.EncodeToString(buf.Bytes())
}

func TestValidateRequest(t *testing.T) {
	t.Parallel()
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
		{Sender: "Alex", Game: "other", Payload: payload, Version: protocolVersion},
		{Sender: "Alex", Game: "lethal-company", Payload: payload, Version: protocolVersion},
		{Sender: "Alex", Game: "stardew", Payload: payload, Version: "1"},
	} {
		if _, err := validateRequest(request); err == nil {
			t.Fatalf("validateRequest(%+v) accepted invalid input", request)
		}
	}
}

func TestValidateRequestCapsPayload(t *testing.T) {
	t.Parallel()
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

// pairedService is a LAN service with its own data folder, served over loopback.
func pairedService(t *testing.T, items *store.Store, emit func(string, any)) (*Service, string) {
	t.Helper()
	service := NewService(Deps{Store: items, Dir: t.TempDir(), Emit: emit})
	server := httptest.NewServer(service.handler())
	t.Cleanup(server.Close)
	service.mu.Lock()
	service.enabled = true
	service.listener = server.Listener
	service.mu.Unlock()
	return service, strings.TrimPrefix(server.URL, "http://")
}

// pair has joiner enter the code that host shows.
func pair(t *testing.T, host, joiner *Service, hostAddr string) {
	t.Helper()
	code, err := host.PairCode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := joiner.Pair(t.Context(), hostAddr, strings.ToLower(code)); err != nil {
		t.Fatal(err)
	}
}

func TestPairingLocksOutAfterWrongCodes(t *testing.T) {
	t.Parallel()
	host, hostAddr := pairedService(t, nil, nil)
	joiner, _ := pairedService(t, nil, nil)

	code, err := host.PairCode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wrong := "AAAA-AAAA"
	if code == wrong {
		wrong = "BBBB-BBBB"
	}
	// Each attempt spends a full key derivation, so they run side by side.
	var wg sync.WaitGroup
	for range maxFailures {
		wg.Go(func() {
			if err := joiner.Pair(t.Context(), hostAddr, wrong); err == nil {
				t.Error("a wrong code paired")
			}
		})
	}
	wg.Wait()
	if err := joiner.Pair(t.Context(), hostAddr, code); err == nil {
		t.Fatal("a locked-out computer paired")
	}
}

func TestPairing(t *testing.T) {
	t.Parallel()
	host, hostAddr := pairedService(t, nil, nil)
	joiner, _ := pairedService(t, nil, nil)

	code, err := host.PairCode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := joiner.Pair(t.Context(), hostAddr, code); err != nil {
		t.Fatal(err)
	}
	hostID, _ := host.book.self()
	joinerID, _ := joiner.book.self()
	if key := joiner.book.key(hostID); len(key) != 32 || string(key) != string(host.book.key(joinerID)) {
		t.Fatal("both computers must hold the same key")
	}
	if err := joiner.Pair(t.Context(), hostAddr, code); err == nil {
		t.Fatal("a code paired twice")
	}
	info, err := os.Stat(filepath.Join(joiner.book.dir, "peers.json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("peers.json = %v, %v", info, err)
	}
	if err := joiner.Unpair(hostID); err != nil || joiner.book.key(hostID) != nil {
		t.Fatalf("unpair: %v", err)
	}
}

func TestTransferTokenScope(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	senderStore := store.OpenAt(t.TempDir())
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "mod.dll"), []byte("from sender"), 0o600); err != nil {
		t.Fatal(err)
	}
	key := store.NexusKey(7, 2)
	if err := senderStore.AddDir(t.Context(), "stardew", key, source); err != nil {
		t.Fatal(err)
	}
	// An optional file laid over the main file travels as its own store item.
	optional := t.TempDir()
	if err := os.WriteFile(filepath.Join(optional, "alt.png"), []byte("optional"), 0o600); err != nil {
		t.Fatal(err)
	}
	optKey := store.NexusKey(7, 3)
	if err := senderStore.AddDir(t.Context(), "stardew", optKey, optional); err != nil {
		t.Fatal(err)
	}

	// Files from a source other than Nexus travel too.
	ghKey := github.Key("o", "r", "v1", "m.zip")
	if err := senderStore.AddDir(t.Context(), "stardew", ghKey, source); err != nil {
		t.Fatal(err)
	}
	// So does an archive the sender installed from disk, which exists nowhere else; its key names the archive's hash,
	// not the folder's.
	localKey := store.LocalKey(strings.Repeat("cd", 32))
	if err := senderStore.AddDir(t.Context(), "stardew", localKey, source); err != nil {
		t.Fatal(err)
	}

	receiverStore := store.OpenAt(t.TempDir())
	arrivals := make(chan Arrival, 2)
	emit := func(_ string, data any) {
		if arrival, ok := data.(Arrival); ok {
			arrivals <- arrival
		}
	}
	receiver, receiverAddr := pairedService(t, receiverStore, emit)
	sender, _ := pairedService(t, senderStore, nil)

	var payload bytes.Buffer
	if _, err := share.Write(&payload, "stardew", profile.Profile{
		Name: "Farm friends",
		Entries: []profile.Entry{
			{Key: key, Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 2}},
			{Key: optKey, Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 3}, OverlayOf: key, OverlayFrom: "a", OverlayTo: "b", OverlayOff: true},
			{Key: ghKey, Source: profile.Source{Kind: profile.KindGitHub, Repo: "o/r", Tag: "v1", Asset: "m.zip"}},
			{Key: localKey, Source: profile.Source{Kind: profile.KindLocal, Name: "Mine.zip"}},
		},
	}, t.TempDir(), share.Include{DisabledMods: true, LocalFiles: true}); err != nil {
		t.Fatal(err)
	}
	if pv, err := share.ReadBytes(payload.Bytes()); err != nil || pv.Entries[1].Overlay == nil || *pv.Entries[1].Overlay != (share.Overlay{From: "a", To: "b", Off: true}) {
		t.Fatalf("payload overlay = %+v, %v", pv.Entries, err)
	}
	// The payload is built knowing whether the receiver is paired, so archives from disk go only to a paired one.
	var builtPaired []bool
	send := func() Arrival {
		t.Helper()
		build := func(paired bool) ([]byte, error) {
			builtPaired = append(builtPaired, paired)
			return payload.Bytes(), nil
		}
		if err := sender.sendPayload(t.Context(), receiverAddr, "stardew", "", build); err != nil {
			t.Fatal(err)
		}
		select {
		case arrival := <-arrivals:
			return arrival
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for share")
			return Arrival{}
		}
	}
	if send().Paired {
		t.Fatal("an unpaired share received a transfer grant")
	}
	pair(t, receiver, sender, receiverAddr)
	receiver.rateMu.Lock()
	receiver.lastReceive = map[string]time.Time{}
	receiver.rateMu.Unlock()
	arrival := send()
	if !arrival.Paired {
		t.Fatal("a paired share did not receive a transfer grant")
	}
	if !slices.Equal(builtPaired, []bool{false, true}) {
		t.Fatalf("payload built for paired = %v, want unpaired then paired", builtPaired)
	}
	if err := receiver.Transfer(t.Context(), arrival.ID); err != nil {
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
	if src, pkg, _, ok := receiverStore.Meta("stardew", localKey); !ok || src != profile.KindLocal || pkg != "Mine.zip" {
		t.Fatalf("local entry meta = %q %q %v", src, pkg, ok)
	}
	if src, pkg, version, ok := receiverStore.Meta("stardew", ghKey); !ok || src != profile.KindGitHub || pkg != "o/r" || version != "v1" {
		t.Fatalf("github entry meta = %q %q %q %v", src, pkg, version, ok)
	}
	dir, err := receiverStore.Path("stardew", optKey)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := fsx.ReadFile(filepath.Join(dir, "alt.png")); err != nil || string(got) != "optional" {
		t.Fatalf("optional file did not transfer: %q %v", got, err)
	}
}

func TestLethalCompanyProfileOverLAN(t *testing.T) {
	senderStore, modsDir := store.OpenAt(t.TempDir()), t.TempDir()
	packages := []struct{ name, version string }{{"Alice-MoreCompany", "1.2.3"}, {"Bob-LateCompany", "2.0.0"}}
	var entries []profile.Entry
	for _, p := range packages {
		key := store.PackageKey(p.name, p.version)
		files := t.TempDir()
		if err := os.WriteFile(filepath.Join(files, p.name+".dll"), []byte(p.name), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := senderStore.AddDir(t.Context(), "lethal-company", key, files); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(modsDir, key), 0o700); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, profile.Entry{
			Key:    key,
			Source: profile.Source{Kind: profile.KindThunderstore, Name: p.name, Version: p.version},
			Mods:   []profile.Component{{ID: mod.NewID(mod.FormatThunderstore, p.name), Folder: "."}},
		})
	}
	config := filepath.Join(modsDir, entries[0].Key, "config.json")
	if err := os.WriteFile(config, []byte(`{"more":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var payload bytes.Buffer
	if _, err := share.Write(&payload, "lethal-company", profile.Profile{Name: "Lobby", Entries: entries}, modsDir); err != nil {
		t.Fatal(err)
	}

	receiverStore := store.OpenAt(t.TempDir())
	arrivals := make(chan Arrival, 1)
	receiver, receiverAddr := pairedService(t, receiverStore, func(_ string, data any) {
		if arrival, ok := data.(Arrival); ok {
			arrivals <- arrival
		}
	})
	sender, _ := pairedService(t, senderStore, nil)
	pair(t, receiver, sender, receiverAddr)
	if err := sender.sendPayload(t.Context(), receiverAddr, "lethal-company", "", fixed(payload.Bytes())); err != nil {
		t.Fatal(err)
	}
	arrival := <-arrivals
	if !arrival.Paired {
		t.Fatal("a paired share did not receive a transfer grant")
	}
	if err := receiver.Transfer(t.Context(), arrival.ID); err != nil {
		t.Fatal(err)
	}
	for _, p := range packages {
		dir, err := receiverStore.Path("lethal-company", store.PackageKey(p.name, p.version))
		if err != nil {
			t.Fatal(err)
		}
		if got, err := fsx.ReadFile(filepath.Join(dir, p.name+".dll")); err != nil || string(got) != p.name {
			t.Fatalf("%s files = %q, %v", p.name, got, err)
		}
		if src, pkg, version, _ := receiverStore.Meta("lethal-company", store.PackageKey(p.name, p.version)); src != profile.KindThunderstore || pkg != p.name || version != p.version {
			t.Fatalf("%s meta = %q %q %q", p.name, src, pkg, version)
		}
	}
	raw, err := base64.RawStdEncoding.DecodeString(arrival.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if pv, err := share.ReadBytes(raw); err != nil || len(pv.Configs) != 1 || string(pv.Configs[0].Data) != `{"more":true}` {
		t.Fatalf("config did not travel: %+v, %v", pv.Configs, err)
	}
}

func TestLoopbackSendReceive(t *testing.T) {
	t.Parallel()
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

	if err := service.sendPayload(t.Context(), strings.TrimPrefix(server.URL, "http://"), "stardew", "", fixed(payload)); err != nil {
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

func TestASecondShareWithinTheRateLimitIsBusy(t *testing.T) {
	t.Parallel()
	payload, err := base64.RawStdEncoding.DecodeString(testPayload(t))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(Deps{})
	server := httptest.NewServer(service.handler())
	defer server.Close()
	peer := strings.TrimPrefix(server.URL, "http://")
	if err := service.sendPayload(t.Context(), peer, "stardew", "", fixed(payload)); err != nil {
		t.Fatal(err)
	}
	err = service.sendPayload(t.Context(), peer, "stardew", "", fixed(payload))
	if !errors.Is(err, ErrPeerBusy) || usererr.KindOf(err) != usererr.Busy {
		t.Fatalf("second send: %v", err)
	}
}

func TestLoopbackLargeMortarRoundTrip(t *testing.T) {
	t.Parallel()
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
			Mods:   []profile.Component{{ID: mod.SMAPI(fmt.Sprintf("mod.%03d", i)), Folder: "."}},
		}
	}
	original := profile.Profile{
		Name:        "Large farm",
		Notes:       "Keep this note.",
		Description: "Profile settings survive LAN transfer.",
		Entries:     entries,
	}
	var archive bytes.Buffer
	if _, err := share.Write(&archive, "stardew", original, modsDir); err != nil {
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
	if err := service.sendPayload(t.Context(), strings.TrimPrefix(server.URL, "http://"), "stardew", "", fixed(archive.Bytes())); err != nil {
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
	t.Parallel()
	if got := peerName(`Pat\ Farmer._mortar._tcp.local.`); got != "Pat Farmer" {
		t.Fatalf("peerName() = %q, want %q", got, "Pat Farmer")
	}
}

func TestPeerNameRejectsOtherServiceTypes(t *testing.T) {
	t.Parallel()
	if got := peerName("OpenThread._meshcop._udp.local."); got != "" {
		t.Fatalf("peerName() = %q, want empty", got)
	}
}

func TestAddPeerDropsOurInstance(t *testing.T) {
	t.Parallel()
	service := NewService(Deps{})
	service.enabled = true
	service.addPeer(&mdns.ServiceEntry{
		Name:       `Pat\ Farmer._mortar._tcp.local.`,
		Port:       1234,
		AddrV4:     net.ParseIP("192.0.2.1"),
		InfoFields: []string{"instance=" + service.instanceID},
	})
	if peers := service.Peers(context.Background()); len(peers) != 0 {
		t.Fatalf("Peers() = %#v, want no peers", peers)
	}
}

func TestPendingKeepsSharesUntilDismissed(t *testing.T) {
	t.Parallel()
	s := &Service{inbox: []Arrival{{ID: 1}, {ID: 2}}, incoming: map[int]incomingTransfer{2: {}}, active: map[int]context.CancelFunc{}}
	first := s.Pending()
	if len(first) != 2 || len(s.Pending()) != 2 {
		t.Fatal("listing the pending shares must not drain them")
	}
	s.Dismiss(1)
	if got := s.Pending(); len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("after dismissing 1: %+v", got)
	}
	if len(s.Inbox()) != 1 || len(s.Pending()) != 0 {
		t.Fatal("the window's Inbox read still drains")
	}
}

func TestPairTranscriptBindsBothNames(t *testing.T) {
	base := pairTranscript([]byte("a"), []byte("b"), "j", "Joiner", "h", "Host")
	for _, other := range [][]byte{
		pairTranscript([]byte("a"), []byte("b"), "j", "Eve", "h", "Host"),
		pairTranscript([]byte("a"), []byte("b"), "j", "Joiner", "h", "Eve"),
	} {
		if bytes.Equal(base, other) {
			t.Fatal("a device name is outside the transcript")
		}
	}
}

func TestEntryMACBindsHashGameAndKey(t *testing.T) {
	key := []byte("pair key")
	mac := entryMAC(key, "stardew", "nexus-1-2", "local-aa")
	for _, other := range []string{
		entryMAC(key, "stardew", "nexus-1-2", "local-bb"),
		entryMAC(key, "stardew", "nexus-1-3", "local-aa"),
		entryMAC(key, "lethal-company", "nexus-1-2", "local-aa"),
		entryMAC([]byte("other"), "stardew", "nexus-1-2", "local-aa"),
	} {
		if other == mac {
			t.Fatal("the MAC does not bind its inputs")
		}
	}
}

func TestExtractTarRefusesEscapesAndLinks(t *testing.T) {
	tarOf := func(h tar.Header, body string) io.Reader {
		var buf bytes.Buffer
		w := tar.NewWriter(&buf)
		h.Size = int64(len(body))
		if err := w.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		return &buf
	}
	for name, h := range map[string]tar.Header{
		"parent":   {Name: "../evil", Typeflag: tar.TypeReg, Mode: 0o600},
		"absolute": {Name: "/evil", Typeflag: tar.TypeReg, Mode: 0o600},
		"symlink":  {Name: "link", Linkname: "/etc", Typeflag: tar.TypeSymlink},
		"hardlink": {Name: "hard", Linkname: "x", Typeflag: tar.TypeLink},
	} {
		root := t.TempDir()
		var total int64
		body := ""
		if h.Typeflag == tar.TypeReg {
			body = "x"
		}
		if err := extractTar(t.Context(), tarOf(h, body), root, &total, func(int64) {}); err == nil {
			t.Errorf("%s: accepted", name)
		}
		if _, err := os.Stat(filepath.Join(root, "..", "evil")); err == nil {
			t.Errorf("%s: wrote outside root", name)
		}
	}
}

// fixed is a payload builder that sends the same bytes whether or not the peer is paired.
func fixed(payload []byte) func(bool) ([]byte, error) {
	return func(bool) ([]byte, error) { return payload, nil }
}

func TestPairedPeerIsExemptFromTheShareRateLimit(t *testing.T) {
	t.Parallel()
	payload, err := base64.RawStdEncoding.DecodeString(testPayload(t))
	if err != nil {
		t.Fatal(err)
	}
	receiver, receiverAddr := pairedService(t, store.OpenAt(t.TempDir()), nil)
	pairedSender, _ := pairedService(t, store.OpenAt(t.TempDir()), nil)
	strangerSender, _ := pairedService(t, store.OpenAt(t.TempDir()), nil)
	pair(t, receiver, pairedSender, receiverAddr)

	for i := range 2 {
		if err := pairedSender.sendPayload(t.Context(), receiverAddr, "stardew", "", fixed(payload)); err != nil {
			t.Fatalf("paired send %d: %v", i+1, err)
		}
	}
	if err := strangerSender.sendPayload(t.Context(), receiverAddr, "stardew", "", fixed(payload)); err != nil {
		t.Fatalf("first unpaired send: %v", err)
	}
	if err := strangerSender.sendPayload(t.Context(), receiverAddr, "stardew", "", fixed(payload)); !errors.Is(err, ErrPeerBusy) {
		t.Fatalf("second unpaired send: %v, want ErrPeerBusy", err)
	}
}

func TestExpiredShareIsAnnouncedAndDropped(t *testing.T) {
	t.Parallel()
	var events []Expired
	s := &Service{
		deps: Deps{Emit: func(name string, data any) {
			if expired, ok := data.(Expired); ok && name == ExpiredEvent {
				events = append(events, expired)
			}
		}},
		inbox:    []Arrival{{ID: 1, Sender: "Alex"}, {ID: 2, Sender: "Sam"}},
		incoming: map[int]incomingTransfer{1: {Sender: "Alex"}, 2: {Sender: "Sam"}},
		active:   map[int]context.CancelFunc{2: func() {}},
	}
	s.expireShare(1)
	s.expireShare(2)
	s.expireShare(1)
	if len(events) != 1 || events[0] != (Expired{ID: 1, Sender: "Alex"}) {
		t.Fatalf("events = %+v, want one for Alex's share", events)
	}
	if got := s.Pending(); len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("pending = %+v, want only the share still transferring", got)
	}
}

func TestPeersReportPairing(t *testing.T) {
	t.Parallel()
	service := NewService(Deps{})
	service.enabled = true
	if err := service.book.add(PairedPeer{ID: "paired-id", Name: "Pat"}, []byte("key")); err != nil {
		t.Fatal(err)
	}
	for i, instance := range []string{"paired-id", "stranger-id"} {
		service.addPeer(&mdns.ServiceEntry{
			Name:       fmt.Sprintf(`Peer%d._mortar._tcp.local.`, i),
			Port:       1234 + i,
			AddrV4:     net.ParseIP("192.0.2.1"),
			InfoFields: []string{"instance=" + instance},
		})
	}
	got := map[string]bool{}
	for _, peer := range service.Peers(context.Background()) {
		got[peer.Name] = peer.Paired
	}
	if want := map[string]bool{"Peer0": true, "Peer1": false}; !maps.Equal(got, want) {
		t.Fatalf("paired = %v, want %v", got, want)
	}
}
