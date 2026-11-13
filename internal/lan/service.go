// Package lan exchanges profile share links with Mortar installations on the local network.
package lan

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
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

	"github.com/Rethunk-Tech/go-safefs"
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
	// maxUnpairedPayloadBytes bounds a share from a sender that has not proved it is paired: it sits in memory for
	// transferTTL, up to maxUnpairedInbox of them, and carries a profile plus its small config files.
	maxUnpairedPayloadBytes = 8 << 20
	maxUnpairedRequestBytes = maxUnpairedPayloadBytes*4/3 + 4096
	rateLimit               = 10 * time.Second
	peerTTL                 = 6 * time.Second
	// browseInterest is how long after a Peers call discovery keeps running; nobody looking means no mDNS traffic.
	browseInterest = 30 * time.Second
	queryTimeout   = 2 * time.Second
	httpTimeout    = 5 * time.Second
	nonceTTL       = time.Minute
	maxNonces      = 1024
	// maxNoncesPerPeer keeps one host from using up the handshakes every sender needs.
	maxNoncesPerPeer = 16
	transferTTL      = 5 * time.Minute
	// maxShareRequests bounds /share bodies held in memory at once; maxUnpairedInbox bounds arrivals nobody paired.
	maxShareRequests = 4
	// senderHeader names the sending device on a /share request so the receiver can rate limit strangers before reading.
	senderHeader     = "X-Mortar-Sender"
	maxUnpairedInbox = 20
	// maxHelloField bounds a /hello id or challenge (both are 43-character tokens) and maxHelloHost its Host (a DNS
	// name plus port), since every handshake record holds all three for a minute.
	maxHelloField = 128
	maxHelloHost  = 256
	// storeIdle is how long a served store entry may make no progress before the connection is dropped; unlike the
	// small request deadlines it bounds silence, not the whole stream, which can run for minutes.
	storeIdle = time.Minute
)

// Arrival is a profile share received from another Mortar installation.
type Arrival struct {
	ID          int    `json:"id"`
	Sender      string `json:"sender"`
	Game        string `json:"game"`
	Payload     string `json:"payload"`
	ProfileName string `json:"profileName"`
	Paired      bool   `json:"paired"`
	// SenderID and ProfileID name the sender's computer and profile, so a later share of the same profile can update
	// what this one made of it. SenderID is set only for a paired sender, whose id is proven.
	SenderID  string `json:"senderId,omitempty"`
	ProfileID string `json:"profileId,omitempty"`
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

	useMu sync.Mutex
	// checkFirewall reads the firewall state; tests replace it.
	checkFirewall func(ctx context.Context, anyAddr bool) (bool, error)
	// networkPublic reports that every connected network is Public; tests replace it.
	networkPublic func() bool
	// rulesAny is the allow-any-address setting the firewall rules were last written for.
	rulesAny bool
	lifeMu   sync.Mutex
	mu       sync.RWMutex
	closed   bool
	enabled  bool
	peers    map[string]peerRecord
	// wantUntil is when discovery stops unless Peers is asked again; wake starts it.
	wantUntil time.Time
	wake      chan struct{}
	inbox     []Arrival
	nextID    int

	book    peerBook
	pairing pairHost

	nonces map[string]nonceRecord
	// salts are the seals already served, so a captured sealed request is not served twice; decoyKey answers /hello
	// for an id this computer is not paired with.
	salts    map[string]time.Time
	decoyKey []byte
	grants   map[string]transferGrant

	outMu        sync.Mutex
	outgoing     map[string]*outgoingState
	nextOutgoing int
	incoming     map[int]incomingTransfer
	active       map[int]context.CancelFunc

	configuredPort int

	server   *http.Server
	listener net.Listener
	mdns     *mdns.Server
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	rateMu      sync.Mutex
	lastReceive map[string]time.Time
	// sharesInFlight counts /share requests being read or decoded.
	sharesInFlight int
	// receiveIdle overrides storeIdle for a receiving copy; tests shorten it.
	receiveIdle time.Duration
	// now is the clock the progress throttle reads; nil means time.Now. A test pins it so a loaded runner's slow
	// unpacking cannot pass a window the test did not open.
	now func() time.Time
	// fetchTried, when set, is called as fetchEntry starts each try; tests count tries with it, since a try can idle
	// out before its request reaches the sender.
	fetchTried func()
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
	SenderPort int    `json:"senderPort"`
	ProfileID  string `json:"profileId,omitempty"`
	// Digests vouches for each store entry the share lists, by entry key; only a paired sender sends them.
	Digests map[string]entryDigest `json:"digests,omitempty"`
}

// NewService returns a LAN sharing service that is disabled until SetEnabled is called.
func NewService(deps Deps) *Service {
	instanceID := randomToken()
	decoy := make([]byte, 32)
	_, _ = rand.Read(decoy)
	service := &Service{
		decoyKey:      decoy,
		deps:          deps,
		checkFirewall: firewallBlocked,
		networkPublic: networkIsPublic,
		name:          localName(),
		instanceID:    instanceID,
		peers:         map[string]peerRecord{},
		wake:          make(chan struct{}, 1),
		lastReceive:   map[string]time.Time{},
		nonces:        map[string]nonceRecord{},
		grants:        map[string]transferGrant{},
		outgoing:      map[string]*outgoingState{},
		incoming:      map[int]incomingTransfer{},
		active:        map[int]context.CancelFunc{},
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
	return s.start(context.Background())
}

// Apply reconciles the listener with the settings. Nothing listens until LAN sharing is first used (ensureStarted),
// so a fresh install never raises a firewall prompt; once it has run, later launches start it at once.
//
//wails:ignore
func (s *Service) Apply() error {
	cfg := s.deps.Settings.Get()
	if !cfg.LanSharing {
		return s.stop()
	}
	if cfg.LanStarted && !s.listening() {
		// Listening with the rules gone makes Windows prompt and write its own block rules, and a prompt at launch
		// has no context for the user: forget the flag instead, so the next LAN use runs the elevated step.
		if blocked, err := s.checkFirewall(context.Background(), cfg.LanAllowAnyAddress); err != nil || blocked {
			slog.Warn("LAN sharing: firewall rules are missing, so it stays off until it is used", "err", err)
			_, uerr := s.deps.Settings.Update(func(v *settings.Settings) { v.LanStarted = false })
			return uerr
		}
	}
	if cfg.LanStarted || s.listening() {
		if !s.listening() {
			if s.publicBlocksListening(cfg.LanAllowAnyAddress) {
				return errPublicNetwork()
			}
			s.mu.Lock()
			s.rulesAny = cfg.LanAllowAnyAddress
			s.mu.Unlock()
		}
		s.refreshRules(cfg.LanAllowAnyAddress)
		return s.SetEnabled(true)
	}
	return nil
}

// refreshRules rewrites the firewall rules when the allow-any-address setting flips under a running listener, once per
// flip: the change is the user's own, so the prompt that follows is expected.
func (s *Service) refreshRules(anyAddr bool) {
	s.useMu.Lock()
	defer s.useMu.Unlock()
	s.mu.Lock()
	changed := s.enabled && s.rulesAny != anyAddr
	if changed {
		s.rulesAny = anyAddr
	}
	s.mu.Unlock()
	if changed {
		if err := fixFirewall(context.Background(), anyAddr); err != nil {
			slog.Warn("LAN sharing firewall", "err", err)
		}
	}
}

// ensureStarted starts the listener on first use, adding the firewall rule first on Windows so the system prompt
// never appears. It keeps the listener up for the session and remembers that LAN sharing is in use.
func (s *Service) ensureStarted(ctx context.Context) error {
	s.useMu.Lock()
	defer s.useMu.Unlock()
	if s.listening() {
		return nil
	}
	if s.deps.Settings == nil || !s.deps.Settings.Get().LanSharing {
		return usererr.Wrap(usererr.Permission, errors.New("LAN sharing is disabled"))
	}
	anyAddr := s.allowAnyAddress()
	if s.publicBlocksListening(anyAddr) {
		return errPublicNetwork()
	}
	blocked, err := firewallBlocked(ctx, anyAddr)
	if err != nil || blocked {
		if err := fixFirewall(ctx, anyAddr); err != nil {
			slog.Warn("LAN sharing firewall", "err", err)
		}
		// Listening without the rule is what makes Windows raise its own prompt, so an unreadable firewall counts
		// as blocked too.
		if blocked, err = firewallBlocked(ctx, anyAddr); err != nil || blocked {
			if err != nil {
				slog.Warn("LAN sharing firewall", "err", err)
			}
			return usererr.New(usererr.Permission, "Mortar could not add its firewall rule, so nearby computers can't reach it")
		}
	}
	if err := s.start(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	s.rulesAny = anyAddr
	s.mu.Unlock()
	_, err = s.deps.Settings.Update(func(v *settings.Settings) { v.LanStarted = true })
	return err
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
	stop := s.stopLocked()
	s.mu.Unlock()
	stop()
	s.wg.Wait()
}

func (s *Service) start(ctx context.Context) error {
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
	listener, err := config.Listen(ctx, "tcp", address)
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
	browseCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	server := newHTTPServer(s.handler())

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		cancel()
		_ = listener.Close()
		_ = advertiser.Shutdown()
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
		if err := server.Serve(&lanListener{Listener: listener, allowAny: s.allowAnyAddress}); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Warn("LAN sharing server", "err", err)
		}
	}()
	go func() {
		defer s.wg.Done()
		s.browse(browseCtx)
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
	stop := s.stopLocked()
	s.mu.Unlock()
	stop()
	s.wg.Wait()
	return nil
}

// stopLocked turns sharing off and detaches its server, listener and advertiser; the caller unlocks s.mu and then
// runs stop, so shutting them down waits on no lock.
func (s *Service) stopLocked() (stop func()) {
	s.enabled = false
	server, listener, advertiser, cancel := s.server, s.listener, s.mdns, s.cancel
	s.server, s.listener, s.mdns, s.cancel = nil, nil, nil, nil
	s.peers = map[string]peerRecord{}
	s.configuredPort = 0
	return func() { stopResources(server, listener, advertiser, cancel) }
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
func (s *Service) Peers(ctx context.Context) ([]Peer, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
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
	slices.SortStableFunc(out, func(a, b Peer) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// Send sends a profile's .mortar payload to a discovered peer.
func (s *Service) Send(ctx context.Context, peerID, game, profileID string, include share.Include) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	if s.deps.Shares == nil {
		return errors.New("LAN sharing is unavailable")
	}
	return s.sendPayload(ctx, peerID, game, profileID, func(paired bool) ([]byte, error) {
		inc := include
		// Archives installed from disk exist nowhere else, so they go only to a paired computer, which copies them.
		inc.LocalFiles = paired
		// Config files can hold keys and tokens, so only a computer the player paired gets them.
		inc.ConfigFiles = inc.ConfigFiles && paired
		payload, _, err := s.deps.Shares.ExportBytes(game, profileID, inc)
		return payload, err
	})
}

// sendPayload sends the payload build makes, told whether the peer is paired with this computer.
func (s *Service) sendPayload(ctx context.Context, peerID, game, profileID string, build func(paired bool) ([]byte, error)) (err error) {
	name := s.peerName(peerID)
	defer func() {
		if err != nil {
			slog.Warn("lan: send failed", "peer", name, "err", err)
		}
	}()
	hello, err := s.hello(ctx, peerID)
	if err != nil {
		return err
	}
	// A peer listed as paired (by the instance it advertised) must answer as that paired id.
	s.mu.Lock()
	record, listed := s.peers[peerID]
	s.mu.Unlock()
	if listed && hello.ID != record.instance && s.book.key(record.instance) != nil {
		return errUnproven
	}
	key, err := s.provenKey(hello)
	if err != nil {
		return err
	}
	var session []byte
	if key != nil {
		selfID, err := s.book.self()
		if err != nil {
			return err
		}
		session = sessionKey(key, hello.challenge, hello.Nonce, selfID, hello.ID, hello.host)
	}
	payload, err := build(key != nil)
	if err != nil {
		return err
	}
	encoded := base64.RawStdEncoding.EncodeToString(payload)
	shared, err := validateRequest(shareRequest{Sender: s.deviceName(), Game: game, Payload: encoded, Version: protocolVersion})
	if err != nil {
		return err
	}
	slog.Info("lan: sending a profile", "profile", shared.Name, "game", game, "peer", name, "paired", key != nil, "files", len(transferItems(shared)), "bytes", len(payload))
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
		SenderPort: s.port(),
		ProfileID:  profileID,
	}
	if key != nil {
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
	var resp *http.Response
	if session != nil {
		resp, err = sealedDo(ctx, http.MethodPost, endpoint, session, "", bytes.NewReader(body), "application/json", senderHeader, selfID, nonceHeader, hello.Nonce)
	} else {
		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("prepare profile share: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(senderHeader, selfID)
		resp, err = peerClient.Do(req)
	}
	if err != nil {
		return usererr.Wrap(usererr.Network, fmt.Errorf("%s could not be reached: %w", name, err))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests {
		return usererr.Wrap(usererr.Busy, ErrPeerBusy)
	}
	refused := func(text string) error {
		return usererr.New(usererr.Invalid, fmt.Sprintf("%s refused the profile: %s", name, text))
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		if text := strings.TrimSpace(string(message)); text != "" {
			return refused(text)
		}
		return refused(fmt.Sprintf("HTTP %d", resp.StatusCode))
	}
	var result shareResponse
	if err := json.NewDecoder(safefs.CapReader(resp.Body, 64<<10)).Decode(&result); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read profile share response: %w", err)
	}
	if session != nil && result.Paired && result.TransferToken != "" {
		// The receiver names the files it will pull, but only files this share lists may be granted.
		listed := transferItems(shared)
		granted := slices.DeleteFunc(slices.Clone(result.EntryKeys), func(k string) bool {
			return !slices.ContainsFunc(listed, func(item transferItem) bool { return item.Key == k })
		})
		tk := transferKey(session, result.TransferToken)
		s.rememberGrant(result.TransferToken, game, granted, tk)
		s.startOutgoing(result.TransferToken, game, name, shared.Name, len(granted), tk)
	}
	slog.Info("lan: sent a profile", "peer", name, "profile", shared.Name, "paired", result.Paired)
	s.rememberAddress(peerID)
	return nil
}

// entryDigests hashes the listed entries the store holds and vouches for each with the pair key.
func (s *Service) entryDigests(key []byte, game string, items []transferItem) map[string]entryDigest {
	out := make(map[string]entryDigest, len(items))
	for _, item := range items {
		// A backup's item was vouched for only by the backup; the receiver downloads it from its source instead.
		if s.deps.Store.Unverified(game, item.Key) {
			continue
		}
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

// provenKey returns the pairing key for the peer that answered, or nil when it is not paired or did not prove it
// holds the key.
func (s *Service) provenKey(hello helloResponse) ([]byte, error) {
	key := s.book.key(hello.ID)
	if key == nil {
		return nil, nil
	}
	selfID, err := s.book.self()
	if err != nil {
		return nil, err
	}
	if hello.challenge == "" || hello.Proof == "" || !hmac.Equal([]byte(helloProof(key, hello.challenge, selfID, hello.ID, hello.host)), []byte(hello.Proof)) {
		return nil, errUnproven
	}
	return key, nil
}

// errUnproven refuses a send to a computer that claims a paired id without proving it holds that pairing's key:
// sending it unsealed instead would hand a paired profile to whoever answered.
var errUnproven = errors.New("that computer did not prove it is the paired computer; pair the two again if one was reinstalled")

func (s *Service) hello(ctx context.Context, peerID string) (helloResponse, error) {
	endpoint, err := shareEndpoint(peerID, "/hello")
	if err != nil {
		return helloResponse{}, err
	}
	challenge := randomToken()
	selfID, err := s.book.self()
	if err != nil {
		return helloResponse{}, err
	}
	endpoint += "?" + url.Values{"challenge": {challenge}, "id": {selfID}}.Encode()
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return helloResponse{}, fmt.Errorf("prepare LAN handshake: %w", err)
	}
	resp, err := peerClient.Do(req)
	if err != nil {
		return helloResponse{}, usererr.Wrap(usererr.Network, fmt.Errorf("contact LAN peer: %w", err))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return helloResponse{}, usererr.New(usererr.Network, fmt.Sprintf("LAN peer handshake failed: HTTP %d", resp.StatusCode))
	}
	var hello helloResponse
	if err := json.NewDecoder(safefs.CapReader(resp.Body, 16<<10)).Decode(&hello); err != nil {
		return helloResponse{}, fmt.Errorf("read LAN peer handshake: %w", err)
	}
	hello.challenge = challenge
	hello.host, _ = peerAddress(peerID)
	if hello.Nonce == "" {
		return helloResponse{}, usererr.Wrap(usererr.Damaged, errors.New("LAN peer returned no handshake nonce"))
	}
	if hello.Version != protocolVersion {
		return helloResponse{}, usererr.Wrap(usererr.Outdated, errors.New("update Mortar on both computers"))
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

// newHTTPServer has no blanket read or write timeout: those would cut a long store stream, so each handler sets its
// own deadline.
func newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{Handler: handler, ReadHeaderTimeout: httpTimeout, IdleTimeout: httpTimeout}
}

func (s *Service) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", withDeadline(s.handleHello))
	mux.HandleFunc("POST /share", jsonOnly(withDeadline(s.sealed(s.resolveShare, true, s.handleShare))))
	mux.HandleFunc("GET /store/", s.sealed(s.resolveGrant, false, s.handleStore))
	mux.HandleFunc("POST /outgoing", jsonOnly(withDeadline(s.sealed(s.resolveOutgoing, false, s.handleOutgoing))))
	mux.HandleFunc("POST "+pairBeginPath, jsonOnly(withDeadline(s.handlePairBegin)))
	mux.HandleFunc("POST "+pairFinishPath, jsonOnly(withDeadline(s.handlePairFinish)))
	// An older Mortar pairs on these routes and shows whatever body a refusal carries.
	for _, old := range []string{"POST /pair/begin", "POST /pair/finish"} {
		mux.HandleFunc(old, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "update Mortar on both computers to pair", http.StatusNotFound)
		})
	}
	return mux
}

// jsonOnly refuses a request that is not declared JSON. A web page can send text/plain across origins without a preflight,
// so the declaration is what keeps a visited page from posting to the listener.
func jsonOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
			http.Error(w, "content type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
		next(w, r)
	}
}

// withDeadline bounds a small request's read and write, which the server no longer does for every handler; the
// deadlines are cleared after, as the connection may serve a longer request next.
func withDeadline(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)
		_ = rc.SetReadDeadline(time.Now().Add(httpTimeout))
		_ = rc.SetWriteDeadline(time.Now().Add(httpTimeout))
		defer func() {
			_ = rc.SetReadDeadline(time.Time{})
			_ = rc.SetWriteDeadline(time.Time{})
		}()
		next(w, r)
	}
}

func (s *Service) handleHello(w http.ResponseWriter, r *http.Request) {
	nonce := randomToken()
	now := time.Now()
	peer := remotePeer(r)
	quota := quotaPeer(peer)
	query := r.URL.Query()
	callerID, challenge := query.Get("id"), query.Get("challenge")
	if len(callerID) > maxHelloField || len(challenge) > maxHelloField || len(r.Host) > maxHelloHost {
		http.Error(w, "handshake field is too long", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	fromPeer := 0
	for value, record := range s.nonces {
		if now.After(record.expires) {
			delete(s.nonces, value)
		} else if quotaPeer(record.peer) == quota {
			fromPeer++
		}
	}
	if len(s.nonces) >= maxNonces || fromPeer >= maxNoncesPerPeer {
		s.mu.Unlock()
		http.Error(w, "too many handshakes", http.StatusTooManyRequests)
		return
	}
	s.nonces[nonce] = nonceRecord{peer: peer, expires: now.Add(nonceTTL), challenge: challenge, callerID: callerID, host: r.Host}
	s.mu.Unlock()
	id, err := s.book.self()
	if err != nil {
		http.Error(w, "could not create handshake", http.StatusInternalServerError)
		return
	}
	resp := helloResponse{ID: id, Name: s.deviceName(), Version: protocolVersion, Nonce: nonce}
	// Every caller gets a proof; an id this computer is not paired with gets one under a throwaway key, so the
	// answer does not say which ids are paired.
	if challenge != "" {
		key := s.book.key(callerID)
		if key == nil {
			key = s.decoyKey
		}
		resp.Proof = helloProof(key, challenge, callerID, id, r.Host)
	}
	writeJSON(w, resp)
}

// shareBodyLimit is the largest /share body read: a sealed request has proved it is paired, any other gets the
// unpaired cap.
func shareBodyLimit(sealed bool) int64 {
	if sealed {
		return maxRequestBytes
	}
	return maxUnpairedRequestBytes
}

func (s *Service) handleShare(w http.ResponseWriter, r *http.Request) {
	_, isSealed := r.Context().Value(sealedCtxKey{}).(*sealedPeer)
	limit := shareBodyLimit(isSealed)
	if r.ContentLength > limit {
		http.Error(w, "request is too large", http.StatusRequestEntityTooLarge)
		return
	}
	peer := remotePeer(r)
	quota := quotaPeer(peer)
	// A peer inside its rate limit is refused whatever sender it names. Only a sealed request has proved it is paired,
	// so any other is charged before its body is read: the sender header alone is unauthenticated.
	charged := !isSealed
	if s.receiveLimited(quota) || charged && !s.allowReceive(quota) {
		http.Error(w, "profile shares from this peer are temporarily rate limited", http.StatusTooManyRequests)
		return
	}
	if !s.acquireShare() {
		http.Error(w, "too many profile shares at once", http.StatusServiceUnavailable)
		return
	}
	defer s.releaseShare()
	r.Body = http.MaxBytesReader(w, r.Body, limit)
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
	var key, session []byte
	paired := false
	if sealed, ok := r.Context().Value(sealedCtxKey{}).(*sealedPeer); ok && request.SenderPort > 0 {
		// The seal proved the sender holds the pair key; its id comes from the handshake, not from the body.
		request.SenderID, session = sealed.id, sealed.session
		key, paired = s.book.key(sealed.id), true
	}
	if !paired && !charged && !s.allowReceive(quota) {
		http.Error(w, "profile shares from this peer are temporarily rate limited", http.StatusTooManyRequests)
		return
	}
	if !paired && len(request.Payload) > maxUnpairedRequestBytes {
		http.Error(w, errPayloadTooLarge.Error(), http.StatusRequestEntityTooLarge)
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
	slog.Info("lan: received a profile", "profile", strings.ReplaceAll(strings.ReplaceAll(shared.Name, "\n", ""), "\r", ""), "sender", strings.ReplaceAll(strings.ReplaceAll(request.Sender, "\n", ""), "\r", ""), "paired", paired, "files", len(items))
	response := shareResponse{Paired: paired}
	var arrivalTransfer incomingTransfer
	if paired {
		token := randomToken()
		keys := make([]string, len(items))
		for i, item := range items {
			keys[i] = item.Key
		}
		response.TransferToken = token
		response.EntryKeys = keys
		arrivalTransfer = incomingTransfer{
			Sender:  request.Sender,
			Peer:    net.JoinHostPort(peer, strconv.Itoa(request.SenderPort)),
			Game:    request.Game,
			Token:   token,
			Key:     transferKey(session, token),
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
		ProfileID:   request.ProfileID,
	}
	if paired {
		arrival.SenderID = request.SenderID
	}
	s.mu.Lock()
	s.nextID++
	arrival.ID = s.nextID
	s.inbox = append(s.inbox, arrival)
	if arrivalTransfer.Token != "" {
		s.incoming[arrival.ID] = arrivalTransfer
	} else {
		s.dropOldestUnpaired()
	}
	time.AfterFunc(transferTTL, func() { s.expireShare(arrival.ID) })
	s.mu.Unlock()
	if s.deps.Emit != nil {
		s.deps.Emit(ArrivedEvent, arrival)
	}
	writeJSON(w, response)
}

// expireShare drops an incoming share whose transfer window has closed and tells the window, unless the transfer is
// running or the share was already dismissed.
func (s *Service) expireShare(id int) {
	s.mu.Lock()
	_, waiting := s.incoming[id]
	idx := slices.IndexFunc(s.inbox, func(a Arrival) bool { return a.ID == id })
	_, running := s.active[id]
	// A paired share lives while its transfer window does; an unpaired one has no window and ages out of the inbox.
	if idx < 0 || running || (!waiting && s.inbox[idx].Paired) {
		s.mu.Unlock()
		return
	}
	delete(s.incoming, id)
	sender := s.inbox[idx].Sender
	s.inbox = slices.Delete(s.inbox, idx, idx+1)
	s.mu.Unlock()
	if s.deps.Emit != nil {
		s.deps.Emit(ExpiredEvent, Expired{ID: id, Sender: sender})
	}
}

// dropOldestUnpaired keeps the unpaired arrivals within maxUnpairedInbox; the caller holds s.mu.
func (s *Service) dropOldestUnpaired() {
	unpaired := 0
	for _, a := range s.inbox {
		if !a.Paired {
			unpaired++
		}
	}
	for unpaired > maxUnpairedInbox {
		i := slices.IndexFunc(s.inbox, func(a Arrival) bool { return !a.Paired })
		s.inbox = slices.Delete(s.inbox, i, i+1)
		unpaired--
	}
}

func (s *Service) acquireShare() bool {
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	if s.sharesInFlight >= maxShareRequests {
		return false
	}
	s.sharesInFlight++
	return true
}

func (s *Service) releaseShare() {
	s.rateMu.Lock()
	s.sharesInFlight--
	s.rateMu.Unlock()
}

// resolveShare finds the session key of a sealed share from the handshake its nonce names. The nonce is spent
// only after the address now asking is the one it was handed to.
func (s *Service) resolveShare(r *http.Request) ([]byte, string, *sealedPeer, error) {
	id, nonce := r.Header.Get(senderHeader), r.Header.Get(nonceHeader)
	s.mu.Lock()
	record, ok := s.nonces[nonce]
	if !ok || record.peer != remotePeer(r) {
		s.mu.Unlock()
		return nil, "", nil, errors.New("handshake not recognised")
	}
	delete(s.nonces, nonce)
	s.mu.Unlock()
	key := s.book.key(id)
	if key == nil || time.Now().After(record.expires) || record.callerID != id {
		return nil, "", nil, errors.New("handshake not recognised")
	}
	selfID, err := s.book.self()
	if err != nil {
		return nil, "", nil, err
	}
	session := sessionKey(key, record.challenge, nonce, id, selfID, record.host)
	return session, "", &sealedPeer{id: id, session: session}, nil
}

// resolveGrant finds the transfer key of the grant a store pull names.
func (s *Service) resolveGrant(r *http.Request) ([]byte, string, *sealedPeer, error) {
	handle := r.Header.Get(handleHeader)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for token, grant := range s.grants {
		if tokenHandle(token) == handle && grant.Key != nil {
			return grant.Key, token, nil, nil
		}
	}
	return nil, "", nil, errors.New("no such grant")
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
		return share.Shared{}, usererr.Wrap(usererr.Invalid, errors.New("game is invalid"))
	}
	if request.Version != protocolVersion {
		return share.Shared{}, usererr.Wrap(usererr.Outdated, errors.New("update Mortar on both computers"))
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

// receiveLimited reports whether peer is still inside the window a charge opened, without charging it.
func (s *Service) receiveLimited(peer string) bool {
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	at, ok := s.lastReceive[peer]
	return ok && time.Now().Before(at.Add(rateLimit))
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

// quotaPeer is the rate-limit and slot key for host: an IPv4 address itself, an IPv6 address as its /64. Privacy
// addresses on one LAN otherwise each get their own handshake, pairing and guess budget.
func quotaPeer(host string) string {
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	ip = ip.To16()
	if ip == nil {
		return host
	}
	masked := make(net.IP, net.IPv6len)
	copy(masked, ip)
	clear(masked[8:])
	return masked.String()
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
		case err := <-done:
			if err != nil {
				slog.Debug("mdns query", "err", err)
			}
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
			if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
				slog.Debug("mdns query", "err", err)
			}
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

func (s *Service) allowAnyAddress() bool {
	return s.deps.Settings != nil && s.deps.Settings.Get().LanAllowAnyAddress
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

func postRaw(ctx context.Context, endpoint string, body []byte) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		cancel()
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := peerClient.Do(req)
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

// peerName is the name a discovered peer goes by, or its address when it is no longer listed.
func (s *Service) peerName(peerID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if record, ok := s.peers[peerID]; ok && record.peer.Name != "" {
		return record.peer.Name
	}
	return peerID
}
