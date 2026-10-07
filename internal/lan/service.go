// Package lan exchanges profile share links with Mortar installations on the local network.
package lan

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/hashicorp/mdns"
)

const (
	// ArrivedEvent carries an incoming LAN profile share to the window.
	ArrivedEvent = "lan:arrived"
	// ExpiredEvent tells the window an incoming share ran out its transfer window before it was taken.
	ExpiredEvent = "lan:expired"

	serviceType     = "_mortar._tcp"
	maxPayloadBytes = share.MaxFileBytes
	maxRequestBytes = maxPayloadBytes*4/3 + 4096
	rateLimit       = 10 * time.Second
	peerTTL         = 6 * time.Second
	// browseInterest is how long after a Peers call discovery keeps running; nobody looking means no mDNS traffic.
	browseInterest = 30 * time.Second
	queryTimeout   = 2 * time.Second
	httpTimeout    = 5 * time.Second
	nonceTTL       = time.Minute
	transferTTL    = 5 * time.Minute
)

// Arrival is a profile share received from another Mortar installation.
type Arrival struct {
	ID          int    `json:"id"`
	Sender      string `json:"sender"`
	Game        string `json:"game"`
	Payload     string `json:"payload"`
	ProfileName string `json:"profileName"`
	Paired      bool   `json:"paired"`
}

// Expired names an incoming share that can no longer be transferred.
type Expired struct {
	ID     int    `json:"id"`
	Sender string `json:"sender"`
}

// Peer is a nearby Mortar installation that can receive a profile share.
type Peer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Paired reports whether this computer holds a pairing key for the peer, so a send carries mod files.
	Paired bool `json:"paired"`
}

// Deps connects LAN sharing to the rest of Mortar.
type Deps struct {
	Shares   *sharesvc.Service
	Settings *settings.Store
	Store    *store.Store
	Version  string
	// Dir is the data folder; paired computers are kept in <Dir>/lan/peers.json. Empty keeps them in memory.
	Dir  string
	Emit func(name string, data any)
}

// Service advertises this Mortar installation, discovers peers, and exchanges profile links.
type Service struct {
	deps       Deps
	name       string
	instanceID string

	lifeMu  sync.Mutex
	mu      sync.RWMutex
	closed  bool
	enabled bool
	peers   map[string]peerRecord
	// wantUntil is when discovery stops unless Peers is asked again; wake starts it.
	wantUntil time.Time
	wake      chan struct{}
	inbox     []Arrival
	nextID    int

	book    peerBook
	pairing pairHost

	nonces   map[string]nonceRecord
	grants   map[string]transferGrant
	incoming map[int]incomingTransfer
	active   map[int]context.CancelFunc

	configuredPort int

	server   *http.Server
	listener net.Listener
	mdns     *mdns.Server
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	rateMu      sync.Mutex
	lastReceive map[string]time.Time
}

type peerRecord struct {
	peer     Peer
	instance string
	lastSeen time.Time
}

type shareRequest struct {
	Sender     string `json:"sender"`
	Game       string `json:"game"`
	Payload    string `json:"payload"`
	Version    string `json:"version"`
	SenderID   string `json:"senderId"`
	Nonce      string `json:"nonce,omitempty"`
	Proof      string `json:"proof,omitempty"`
	SenderPort int    `json:"senderPort"`
	// Digests vouches for each store entry the share lists, by entry key; only a paired sender sends them.
	Digests map[string]entryDigest `json:"digests,omitempty"`
}

// NewService returns a LAN sharing service that is disabled until SetEnabled is called.
func NewService(deps Deps) *Service {
	instanceID, err := randomToken()
	if err != nil {
		instanceID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	service := &Service{
		deps:        deps,
		name:        localName(),
		instanceID:  instanceID,
		peers:       map[string]peerRecord{},
		wake:        make(chan struct{}, 1),
		lastReceive: map[string]time.Time{},
		nonces:      map[string]nonceRecord{},
		grants:      map[string]transferGrant{},
		incoming:    map[int]incomingTransfer{},
		active:      map[int]context.CancelFunc{},
	}
	if deps.Dir != "" {
		service.book.dir = filepath.Join(deps.Dir, "lan")
	}
	// The advertised instance is the stable paired-computer id, so a discovered peer can be matched to the peer book.
	if id, err := service.book.self(); err == nil {
		service.instanceID = id
	}
	return service
}

// Busy reports whether an incoming LAN transfer is active.
//
//wails:ignore
func (s *Service) Busy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.active) > 0
}

// SetEnabled starts or stops the LAN listener and mDNS discovery.
//
//wails:ignore
func (s *Service) SetEnabled(enabled bool) error {
	if !enabled {
		return s.stop()
	}
	desired := s.lanPort()
	s.mu.RLock()
	restart := s.enabled && s.configuredPort != desired
	s.mu.RUnlock()
	if restart {
		if err := s.stop(); err != nil {
			return err
		}
	}
	return s.start()
}

// Shutdown stops LAN sharing permanently as Mortar exits.
//
//wails:ignore
func (s *Service) Shutdown() {
	s.lifeMu.Lock()
	defer s.lifeMu.Unlock()

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.enabled = false
	server, listener, advertiser, cancel := s.resourcesLocked()
	s.clearResourcesLocked()
	s.peers = map[string]peerRecord{}
	s.configuredPort = 0
	s.mu.Unlock()
	stopResources(server, listener, advertiser, cancel)
	s.wg.Wait()
}

func (s *Service) start() error {
	s.lifeMu.Lock()
	defer s.lifeMu.Unlock()

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return errors.New("LAN sharing service is shut down")
	}
	if s.enabled {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	config := net.ListenConfig{}
	address := ":0"
	if configuredPort := s.lanPort(); configuredPort > 0 {
		address = net.JoinHostPort("", strconv.Itoa(configuredPort))
	}
	listener, err := config.Listen(context.Background(), "tcp", address)
	if err != nil {
		return fmt.Errorf("listen for LAN sharing: %w", err)
	}
	tcpAddress, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return errors.New("LAN sharing listener has an unexpected address")
	}
	port := tcpAddress.Port
	zone, err := mdns.NewMDNSService(
		s.deviceName(),
		serviceType,
		"local.",
		"",
		port,
		nil,
		[]string{
			fmt.Sprintf("version=%s", s.deps.Version),
			"instance=" + s.instanceID,
		},
	)
	if err != nil {
		_ = listener.Close()
		return fmt.Errorf("prepare LAN advertisement: %w", err)
	}
	advertiser, err := mdns.NewServer(&mdns.Config{Zone: zone})
	if err != nil {
		_ = listener.Close()
		return fmt.Errorf("advertise LAN sharing: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	server := &http.Server{
		Handler:           s.handler(),
		ReadHeaderTimeout: httpTimeout,
		ReadTimeout:       httpTimeout,
		WriteTimeout:      httpTimeout,
		IdleTimeout:       httpTimeout,
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		stopResources(server, listener, advertiser, cancel)
		return errors.New("LAN sharing service is shut down")
	}
	s.enabled = true
	s.configuredPort = s.lanPort()
	s.server = server
	s.listener = listener
	s.mdns = advertiser
	s.cancel = cancel
	s.mu.Unlock()

	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("LAN sharing server: %v", err)
		}
	}()
	go func() {
		defer s.wg.Done()
		s.browse(ctx)
	}()
	return nil
}

func (s *Service) stop() error {
	s.lifeMu.Lock()
	defer s.lifeMu.Unlock()

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	if !s.enabled {
		s.mu.Unlock()
		return nil
	}
	s.enabled = false
	server, listener, advertiser, cancel := s.resourcesLocked()
	s.clearResourcesLocked()
	s.peers = map[string]peerRecord{}
	s.configuredPort = 0
	s.mu.Unlock()
	stopResources(server, listener, advertiser, cancel)
	s.wg.Wait()
	return nil
}

func (s *Service) resourcesLocked() (*http.Server, net.Listener, *mdns.Server, context.CancelFunc) {
	return s.server, s.listener, s.mdns, s.cancel
}

func (s *Service) clearResourcesLocked() {
	s.server = nil
	s.listener = nil
	s.mdns = nil
	s.cancel = nil
}

func stopResources(server *http.Server, listener net.Listener, advertiser *mdns.Server, cancel context.CancelFunc) {
	if cancel != nil {
		cancel()
	}
	if server != nil {
		ctx, stop := context.WithTimeout(context.Background(), time.Second)
		_ = server.Shutdown(ctx)
		stop()
	}
	if listener != nil {
		_ = listener.Close()
	}
	if advertiser != nil {
		_ = advertiser.Shutdown()
	}
}

// Peers returns the Mortar installations found during the latest discovery rounds. Discovery runs only while
// someone asks: a call after a quiet spell looks for peers before answering, and keeps discovery going for a while.
func (s *Service) Peers(ctx context.Context) []Peer {
	s.mu.Lock()
	cold := !time.Now().Before(s.wantUntil)
	s.wantUntil = time.Now().Add(browseInterest)
	running := s.enabled && s.listener != nil
	s.mu.Unlock()
	if running {
		if cold {
			qctx, cancel := context.WithTimeout(ctx, queryTimeout+time.Second)
			s.query(qctx)
			cancel()
		}
		select {
		case s.wake <- struct{}{}:
		default:
		}
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, record := range s.peers {
		if now.Sub(record.lastSeen) > peerTTL {
			delete(s.peers, id)
		}
	}
	out := make([]Peer, 0, len(s.peers))
	for _, record := range s.peers {
		peer := record.peer
		peer.Paired = s.book.key(record.instance) != nil
		out = append(out, peer)
	}
	slicesSortPeers(out)
	return out
}

// Send sends a profile's .mortar payload to a discovered peer.
func (s *Service) Send(ctx context.Context, peerID, game, profileID string) error {
	s.mu.RLock()
	enabled := s.enabled
	s.mu.RUnlock()
	if !enabled {
		return errors.New("LAN sharing is disabled")
	}
	if s.deps.Shares == nil {
		return errors.New("LAN sharing is unavailable")
	}
	return s.sendPayload(ctx, peerID, game, func(paired bool) ([]byte, error) {
		inc := share.OwnInclude()
		// Archives installed from disk exist nowhere else, so they go only to a paired computer, which copies them.
		inc.LocalFiles = paired
		payload, _, err := s.deps.Shares.ExportBytes(game, profileID, inc)
		return payload, err
	})
}

// sendPayload sends the payload build makes, told whether the peer is paired with this computer.
func (s *Service) sendPayload(ctx context.Context, peerID, game string, build func(paired bool) ([]byte, error)) error {
	hello, err := s.hello(ctx, peerID)
	if err != nil {
		return err
	}
	key := s.book.key(hello.ID)
	payload, err := build(key != nil)
	if err != nil {
		return err
	}
	encoded := base64.RawStdEncoding.EncodeToString(payload)
	shared, err := validateRequest(shareRequest{Sender: s.deviceName(), Game: game, Payload: encoded, Version: protocolVersion})
	if err != nil {
		return err
	}
	selfID, err := s.book.self()
	if err != nil {
		return err
	}
	request := shareRequest{
		SenderID:   selfID,
		Sender:     s.deviceName(),
		Game:       game,
		Payload:    encoded,
		Version:    protocolVersion,
		Nonce:      hello.Nonce,
		SenderPort: s.port(),
	}
	if key != nil {
		request.Proof = hmacProof(key, hello.Nonce, encoded)
		request.Digests = s.entryDigests(key, game, transferItems(shared))
	}
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode profile share: %w", err)
	}
	endpoint, err := shareEndpoint(peerID, "/share")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("prepare profile share: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("send profile share: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests {
		return usererr.Wrap(usererr.Busy, ErrPeerBusy)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		if text := strings.TrimSpace(string(message)); text != "" {
			return fmt.Errorf("peer rejected profile share: %s", text)
		}
		return fmt.Errorf("peer rejected profile share: HTTP %d", resp.StatusCode)
	}
	var result shareResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read profile share response: %w", err)
	}
	if result.Paired && result.TransferToken != "" && proofMatches(key, hello.Nonce, "response|"+result.TransferToken+"|"+strings.Join(result.EntryKeys, ","), result.Proof) {
		s.rememberGrant(result.TransferToken, game, result.EntryKeys)
	}
	s.rememberAddress(peerID)
	return nil
}

// entryDigests hashes the listed entries the store holds and vouches for each with the pair key.
func (s *Service) entryDigests(key []byte, game string, items []transferItem) map[string]entryDigest {
	out := make(map[string]entryDigest, len(items))
	for _, item := range items {
		dir, err := s.deps.Store.Path(game, item.Key)
		if err != nil {
			continue
		}
		hash, err := store.HashDir(dir)
		if err != nil {
			continue
		}
		out[item.Key] = entryDigest{Hash: hash, MAC: entryMAC(key, game, item.Key, hash)}
	}
	return out
}

func (s *Service) hello(ctx context.Context, peerID string) (helloResponse, error) {
	endpoint, err := shareEndpoint(peerID, "/hello")
	if err != nil {
		return helloResponse{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return helloResponse{}, fmt.Errorf("prepare LAN handshake: %w", err)
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return helloResponse{}, fmt.Errorf("contact LAN peer: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return helloResponse{}, fmt.Errorf("LAN peer handshake failed: HTTP %d", resp.StatusCode)
	}
	var hello helloResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&hello); err != nil {
		return helloResponse{}, fmt.Errorf("read LAN peer handshake: %w", err)
	}
	if hello.Nonce == "" {
		return helloResponse{}, errors.New("LAN peer returned no handshake nonce")
	}
	if hello.Version != protocolVersion {
		return helloResponse{}, errors.New("update Mortar on the other computer")
	}
	return hello, nil
}

func shareEndpoint(peerID, endpointPath string) (string, error) {
	address, err := peerAddress(peerID)
	if err != nil {
		return "", err
	}
	return (&url.URL{Scheme: "http", Host: address, Path: endpointPath}).String(), nil
}

func peerAddress(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme != "http" || parsed.Host == "" || parsed.User != nil {
			return "", errors.New("invalid LAN peer address")
		}
		raw = parsed.Host
	}
	if strings.ContainsAny(raw, "/?#") {
		return "", errors.New("invalid LAN peer address")
	}
	host, portText, err := net.SplitHostPort(raw)
	if err != nil || host == "" {
		return "", errors.New("LAN peer address must be host:port")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return "", errors.New("LAN peer port must be between 1 and 65535")
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}

func (s *Service) rememberAddress(raw string) {
	if s.deps.Settings == nil {
		return
	}
	address, err := peerAddress(raw)
	if err != nil {
		return
	}
	next, err := s.deps.Settings.Update(func(next *settings.Settings) {
		addresses := make([]string, 0, settings.MaxLanAddresses)
		addresses = append(addresses, address)
		for _, current := range next.LanAddresses {
			if current != address && len(addresses) < settings.MaxLanAddresses {
				addresses = append(addresses, current)
			}
		}
		next.LanAddresses = addresses
	})
	if err == nil && s.deps.Emit != nil {
		s.deps.Emit(settings.ChangedEvent, next)
	}
}

func (s *Service) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", s.handleHello)
	mux.HandleFunc("/share", s.handleShare)
	mux.HandleFunc("/store/", s.handleStore)
	mux.HandleFunc("/pair/begin", s.handlePairBegin)
	mux.HandleFunc("/pair/finish", s.handlePairFinish)
	return mux
}

func (s *Service) handleHello(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	nonce, err := randomToken()
	if err != nil {
		http.Error(w, "could not create handshake", http.StatusInternalServerError)
		return
	}
	now := time.Now()
	peer := remotePeer(r)
	s.mu.Lock()
	for value, record := range s.nonces {
		if now.After(record.expires) {
			delete(s.nonces, value)
		}
	}
	s.nonces[nonce] = nonceRecord{peer: peer, expires: now.Add(nonceTTL)}
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	id, err := s.book.self()
	if err != nil {
		http.Error(w, "could not create handshake", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(helloResponse{ID: id, Name: s.deviceName(), Version: protocolVersion, Nonce: nonce})
}

func (s *Service) handleShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.ContentLength > maxRequestBytes {
		http.Error(w, "request is too large", http.StatusRequestEntityTooLarge)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "request is too large", http.StatusRequestEntityTooLarge)
		return
	}
	request, err := decodeRequest(body)
	if err != nil {
		http.Error(w, "invalid profile share", http.StatusBadRequest)
		return
	}
	shared, err := validateRequest(request)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errPayloadTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		http.Error(w, err.Error(), status)
		return
	}
	peer := remotePeer(r)
	key := s.book.key(request.SenderID)
	paired := request.SenderPort > 0 && s.consumeProof(peer, request) && proofMatches(key, request.Nonce, request.Payload, request.Proof)
	if !paired && !s.allowReceive(peer) {
		http.Error(w, "profile shares from this peer are temporarily rate limited", http.StatusTooManyRequests)
		return
	}
	items := transferItems(shared)
	if paired {
		for i, item := range items {
			if d, ok := request.Digests[item.Key]; ok && hmac.Equal([]byte(entryMAC(key, request.Game, item.Key, d.Hash)), []byte(d.MAC)) {
				items[i].Hash = d.Hash
			}
		}
	}
	response := shareResponse{Paired: paired}
	var arrivalTransfer incomingTransfer
	if paired {
		token, tokenErr := randomToken()
		if tokenErr != nil {
			http.Error(w, "could not create transfer token", http.StatusInternalServerError)
			return
		}
		keys := make([]string, len(items))
		for i, item := range items {
			keys[i] = item.Key
		}
		response.TransferToken = token
		response.EntryKeys = keys
		response.Proof = responseProof(key, request.Nonce, token, keys)
		arrivalTransfer = incomingTransfer{
			Sender:  request.Sender,
			Peer:    net.JoinHostPort(peer, strconv.Itoa(request.SenderPort)),
			Game:    request.Game,
			Token:   token,
			Items:   items,
			Expires: time.Now().Add(transferTTL),
		}
	}
	arrival := Arrival{
		Sender:      request.Sender,
		Game:        request.Game,
		Payload:     request.Payload,
		ProfileName: shared.Name,
		Paired:      paired,
	}
	s.mu.Lock()
	s.nextID++
	arrival.ID = s.nextID
	s.inbox = append(s.inbox, arrival)
	if arrivalTransfer.Token != "" {
		s.incoming[arrival.ID] = arrivalTransfer
		time.AfterFunc(transferTTL, func() { s.expireShare(arrival.ID) })
	}
	s.mu.Unlock()
	if s.deps.Emit != nil {
		s.deps.Emit(ArrivedEvent, arrival)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// expireShare drops an incoming share whose transfer window has closed and tells the window, unless the transfer is
// running or the share was already dismissed.
func (s *Service) expireShare(id int) {
	s.mu.Lock()
	incoming, waiting := s.incoming[id]
	_, running := s.active[id]
	if !waiting || running {
		s.mu.Unlock()
		return
	}
	delete(s.incoming, id)
	s.inbox = slices.DeleteFunc(s.inbox, func(a Arrival) bool { return a.ID == id })
	s.mu.Unlock()
	if s.deps.Emit != nil {
		s.deps.Emit(ExpiredEvent, Expired{ID: id, Sender: incoming.Sender})
	}
}

func (s *Service) consumeProof(peer string, request shareRequest) bool {
	if request.Nonce == "" {
		return false
	}
	now := time.Now()
	s.mu.Lock()
	record, ok := s.nonces[request.Nonce]
	delete(s.nonces, request.Nonce)
	s.mu.Unlock()
	return ok && record.peer == peer && now.Before(record.expires)
}

// Inbox returns profile shares received before the window started listening.
func (s *Service) Inbox() []Arrival {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Arrival(nil), s.inbox...)
	s.inbox = nil
	return out
}

// Pending lists the shares waiting for an answer without taking them off the list; Inbox is the window's one-shot
// read.
func (s *Service) Pending() []Arrival {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Arrival(nil), s.inbox...)
}

// Dismiss drops a waiting share and stops its transfer, if one is running.
func (s *Service) Dismiss(id int) {
	s.CancelTransfer(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inbox = slices.DeleteFunc(s.inbox, func(a Arrival) bool { return a.ID == id })
	delete(s.incoming, id)
}

func decodeRequest(body []byte) (shareRequest, error) {
	var request shareRequest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return shareRequest{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return shareRequest{}, errors.New("request has trailing data")
	}
	return request, nil
}

var (
	errPayloadTooLarge = errors.New("profile share payload is too large")
	errInvalidPayload  = errors.New("profile share payload is invalid")
)

func validateRequest(request shareRequest) (share.Shared, error) {
	if request.Sender == "" || len(request.Sender) > 255 || request.Sender != strings.TrimSpace(request.Sender) ||
		strings.ContainsFunc(request.Sender, unicode.IsControl) {
		return share.Shared{}, errors.New("sender is invalid")
	}
	if game.Find(request.Game) == nil {
		return share.Shared{}, errors.New("game is invalid")
	}
	if request.Version != protocolVersion {
		return share.Shared{}, errors.New("update Mortar on the other computer")
	}
	raw, err := base64.RawStdEncoding.DecodeString(request.Payload)
	if err != nil {
		return share.Shared{}, errInvalidPayload
	}
	if len(raw) > maxPayloadBytes {
		return share.Shared{}, errPayloadTooLarge
	}
	if request.Payload == "" || request.Payload != strings.TrimSpace(request.Payload) {
		return share.Shared{}, errInvalidPayload
	}
	pv, err := share.ReadBytes(raw)
	if err != nil {
		return share.Shared{}, fmt.Errorf("%w: %w", errInvalidPayload, err)
	}
	if pv.Game != request.Game {
		return share.Shared{}, errors.New("game does not match the profile")
	}
	return pv.Shared, nil
}

// ErrPeerBusy means the other Mortar took a share from this one moments ago and refuses another for a few seconds.
var ErrPeerBusy = errors.New("the other computer is still taking the last share; try again in a few seconds")

func (s *Service) allowReceive(peer string) bool {
	now := time.Now()
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	for key, at := range s.lastReceive {
		if now.Sub(at) >= rateLimit {
			delete(s.lastReceive, key)
		}
	}
	if at, ok := s.lastReceive[peer]; ok && now.Before(at.Add(rateLimit)) {
		return false
	}
	s.lastReceive[peer] = now
	return true
}

func remotePeer(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	if r.RemoteAddr == "" {
		return "unknown"
	}
	return r.RemoteAddr
}

// browse rediscovers peers every second while Peers was asked recently, and otherwise waits to be woken.
func (s *Service) browse(ctx context.Context) {
	for {
		s.mu.RLock()
		wanted := time.Now().Before(s.wantUntil)
		s.mu.RUnlock()
		if !wanted {
			select {
			case <-ctx.Done():
				return
			case <-s.wake:
			}
			continue
		}
		s.query(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// query runs one mDNS discovery round, adding the peers that answer. The mdns client keeps filling in an entry after
// handing it over, so entries are read only once the round has ended.
func (s *Service) query(ctx context.Context) {
	entries := make(chan *mdns.ServiceEntry, 64)
	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- mdns.QueryContext(queryCtx, &mdns.QueryParam{
			Service: serviceType,
			Domain:  "local",
			Timeout: queryTimeout,
			Entries: entries,
		})
	}()
	var found []*mdns.ServiceEntry
	for {
		select {
		case entry := <-entries:
			if entry != nil {
				found = append(found, entry)
			}
		case <-done:
			for len(entries) > 0 {
				if entry := <-entries; entry != nil {
					found = append(found, entry)
				}
			}
			for _, entry := range found {
				s.addPeer(entry)
			}
			return
		case <-ctx.Done():
			cancel()
			<-done
			return
		}
	}
}

func (s *Service) addPeer(entry *mdns.ServiceEntry) {
	name := peerName(entry.Name)
	if name == "" || entryInstanceID(entry) == s.instanceID {
		return
	}
	address := entryAddress(entry)
	if address == "" || entry.Port < 1 || entry.Port > 65535 {
		return
	}
	id := net.JoinHostPort(address, strconv.Itoa(entry.Port))
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled {
		return
	}
	s.peers[id] = peerRecord{peer: Peer{ID: id, Name: name}, instance: entryInstanceID(entry), lastSeen: time.Now()}
}

func (s *Service) port() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener == nil {
		return 0
	}
	address, ok := s.listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0
	}
	return address.Port
}

func (s *Service) lanPort() int {
	if s.deps.Settings == nil {
		return 0
	}
	port := s.deps.Settings.Get().LanPort
	if port < 0 || port > 65535 {
		return 0
	}
	return port
}

func entryAddress(entry *mdns.ServiceEntry) string {
	if entry.AddrV4 != nil {
		return entry.AddrV4.String()
	}
	if entry.Addr != nil {
		return entry.Addr.String()
	}
	if entry.AddrV6IPAddr != nil {
		return entry.AddrV6IPAddr.IP.String()
	}
	return ""
}

func peerName(name string) string {
	suffix := "." + serviceType + ".local."
	if !strings.HasSuffix(name, suffix) {
		return ""
	}
	name = strings.TrimSuffix(name, suffix)
	return unescapeDNSName(strings.TrimSuffix(name, "."))
}

func entryInstanceID(entry *mdns.ServiceEntry) string {
	for _, field := range entry.InfoFields {
		if instanceID, ok := strings.CutPrefix(field, "instance="); ok {
			return instanceID
		}
	}
	return ""
}

func unescapeDNSName(name string) string {
	var out strings.Builder
	for i := 0; i < len(name); i++ {
		if name[i] != '\\' || i+1 >= len(name) {
			out.WriteByte(name[i])
			continue
		}
		i++
		// \DDD is a decimal byte; anything past 255 is not one, so it stays literal.
		if i+2 < len(name) {
			if value, err := strconv.ParseUint(name[i:i+3], 10, 8); err == nil {
				out.WriteByte(byte(value))
				i += 2
				continue
			}
		}
		out.WriteByte(name[i])
	}
	return out.String()
}

func (s *Service) deviceName() string {
	if s.deps.Settings != nil {
		if name := cleanName(s.deps.Settings.Get().LanName); name != "" {
			return name
		}
	}
	return localName()
}

func localName() string {
	if current, err := user.Current(); err == nil {
		if name := cleanName(current.Name); name != "" {
			return name
		}
	}
	if hostname, err := os.Hostname(); err == nil {
		if name := cleanName(hostname); name != "" {
			return name
		}
	}
	return "Mortar"
}

func cleanName(name string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name))
}

func slicesSortPeers(peers []Peer) {
	for i := 1; i < len(peers); i++ {
		for j := i; j > 0 && peers[j].Name < peers[j-1].Name; j-- {
			peers[j], peers[j-1] = peers[j-1], peers[j]
		}
	}
}

func postRaw(ctx context.Context, endpoint string, body []byte) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		cancel()
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		cancel()
		return nil, err
	}
	resp.Body = cancelOnClose{resp.Body, cancel}
	return resp, nil
}

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c cancelOnClose) Close() error {
	defer c.cancel()
	return c.ReadCloser.Close()
}
