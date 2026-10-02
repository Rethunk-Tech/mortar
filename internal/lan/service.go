// Package lan exchanges profile share links with Mortar installations on the local network.
package lan

import (
	"bytes"
	"context"
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
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/Rethunk-AI/mortar/internal/share"
	"github.com/Rethunk-AI/mortar/internal/sharesvc"
	"github.com/hashicorp/mdns"
)

const (
	// ArrivedEvent carries an incoming LAN profile share to the window.
	ArrivedEvent = "lan:arrived"

	serviceType     = "_mortar._tcp"
	maxPayloadBytes = 1 << 20
	rateLimit       = 10 * time.Second
	peerTTL         = 6 * time.Second
	httpTimeout     = 5 * time.Second
)

// Arrival is a profile share received from another Mortar installation.
type Arrival struct {
	Sender      string `json:"sender"`
	Game        string `json:"game"`
	Payload     string `json:"payload"`
	ProfileName string `json:"profileName"`
}

// Peer is a nearby Mortar installation that can receive a profile share.
type Peer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Deps connects LAN sharing to the rest of Mortar.
type Deps struct {
	Shares  *sharesvc.Service
	Version string
	Emit    func(name string, data any)
}

// Service advertises this Mortar installation, discovers peers, and exchanges profile links.
type Service struct {
	deps Deps
	name string

	lifeMu  sync.Mutex
	mu      sync.RWMutex
	closed  bool
	enabled bool
	peers   map[string]peerRecord
	inbox   []Arrival

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
	lastSeen time.Time
}

type shareRequest struct {
	Sender  string `json:"sender"`
	Game    string `json:"game"`
	Payload string `json:"payload"`
}

// NewService returns a LAN sharing service that is disabled until SetEnabled is called.
func NewService(deps Deps) *Service {
	return &Service{
		deps:        deps,
		name:        localName(),
		peers:       map[string]peerRecord{},
		lastReceive: map[string]time.Time{},
	}
}

// SetEnabled starts or stops the LAN listener and mDNS discovery.
func (s *Service) SetEnabled(enabled bool) error {
	if enabled {
		return s.start()
	}
	return s.stop()
}

// Shutdown stops LAN sharing permanently as Mortar exits.
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
	listener, err := config.Listen(context.Background(), "tcp", ":0")
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
		s.name,
		serviceType,
		"local.",
		"",
		port,
		nil,
		[]string{fmt.Sprintf("version=%s", s.deps.Version)},
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

// Peers returns the Mortar installations found during the latest discovery rounds.
func (s *Service) Peers() []Peer {
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
		out = append(out, record.peer)
	}
	slicesSortPeers(out)
	return out
}

// Send sends a profile's share link to a discovered peer.
func (s *Service) Send(peerID, game, profileID string) error {
	if s.deps.Shares == nil {
		return errors.New("LAN sharing is unavailable")
	}
	info, err := s.deps.Shares.Share(game, profileID, nil)
	if err != nil {
		return err
	}
	if info.App == "" {
		return errors.New("this profile is too large to send as a link")
	}
	payload, ok := strings.CutPrefix(info.App, "mortar://stardew/p/")
	if !ok || payload == "" {
		return errors.New("could not build a profile share link")
	}
	return s.sendPayload(peerID, game, payload)
}

func (s *Service) sendPayload(peerID, game, payload string) error {
	if _, err := validateRequest(shareRequest{Sender: s.name, Game: game, Payload: payload}); err != nil {
		return err
	}
	body, err := json.Marshal(shareRequest{Sender: s.name, Game: game, Payload: payload})
	if err != nil {
		return fmt.Errorf("encode profile share: %w", err)
	}
	endpoint, err := shareEndpoint(peerID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
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
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		if text := strings.TrimSpace(string(message)); text != "" {
			return fmt.Errorf("peer rejected profile share: %s", text)
		}
		return fmt.Errorf("peer rejected profile share: HTTP %d", resp.StatusCode)
	}
	return nil
}

func shareEndpoint(peerID string) (string, error) {
	if !strings.Contains(peerID, "://") {
		peerID = "http://" + peerID
	}
	parsed, err := url.Parse(peerID)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" || parsed.User != nil {
		return "", errors.New("invalid LAN peer address")
	}
	parsed.Path = "/share"
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func (s *Service) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/share", s.handleShare)
	return mux
}

func (s *Service) handleShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.ContentLength > maxPayloadBytes+4096 {
		http.Error(w, "request is too large", http.StatusRequestEntityTooLarge)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadBytes+4096)
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
	if !s.allowReceive(remotePeer(r)) {
		http.Error(w, "profile shares from this peer are temporarily rate limited", http.StatusTooManyRequests)
		return
	}
	arrival := Arrival{
		Sender:      request.Sender,
		Game:        request.Game,
		Payload:     request.Payload,
		ProfileName: shared.Name,
	}
	s.mu.Lock()
	s.inbox = append(s.inbox, arrival)
	s.mu.Unlock()
	if s.deps.Emit != nil {
		s.deps.Emit(ArrivedEvent, arrival)
	}
	w.WriteHeader(http.StatusNoContent)
}

// Inbox returns profile shares received before the window started listening.
func (s *Service) Inbox() []Arrival {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Arrival(nil), s.inbox...)
	s.inbox = nil
	return out
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
	if request.Game != "stardew" {
		return share.Shared{}, errors.New("game is invalid")
	}
	if len(request.Payload) > maxPayloadBytes {
		return share.Shared{}, errPayloadTooLarge
	}
	if request.Payload == "" || request.Payload != strings.TrimSpace(request.Payload) ||
		strings.ContainsAny(request.Payload, "/:#?") {
		return share.Shared{}, errInvalidPayload
	}
	shared, err := share.Parse(request.Payload)
	if err != nil {
		return share.Shared{}, fmt.Errorf("%w: %w", errInvalidPayload, err)
	}
	return shared, nil
}

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

func (s *Service) browse(ctx context.Context) {
	for {
		entries := make(chan *mdns.ServiceEntry, 64)
		queryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		done := make(chan error, 1)
		go func() {
			done <- mdns.QueryContext(queryCtx, &mdns.QueryParam{
				Service: serviceType,
				Domain:  "local",
				Timeout: 2 * time.Second,
				Entries: entries,
			})
		}()
		queryDone := false
		for !queryDone {
			select {
			case entry := <-entries:
				if entry != nil {
					s.addPeer(entry)
				}
			case <-done:
				queryDone = true
			case <-ctx.Done():
				cancel()
				<-done
				return
			}
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func (s *Service) addPeer(entry *mdns.ServiceEntry) {
	name := peerName(entry.Name)
	if name == "" || (name == s.name && entry.Port == s.port()) {
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
	s.peers[id] = peerRecord{peer: Peer{ID: id, Name: name}, lastSeen: time.Now()}
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
	name = strings.TrimSuffix(name, suffix)
	return strings.TrimSuffix(name, ".")
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
