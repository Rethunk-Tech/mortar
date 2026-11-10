package lan

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Totative/go-codepair"
)

// Pairing makes two of the user's own machines trust each other. One shows a short code, the other types it, and both
// keep a 32-byte key for the other; every later share is proved with that key.
//
// Construction: an X25519 exchange (A from the joiner, B from the host) gives a shared secret; each side then proves it
// knows the code with an HMAC over the transcript and that secret, keyed by a PBKDF2 stretch of the code. The long-term
// key is HKDF of the same secret, so it never depends on the code.
//
// Threat model. A passive observer sees A, B and the proofs, but a proof also covers the X25519 secret, which cannot
// be computed without a private key, so the observer cannot test guesses against the code and learns no key. An
// active attacker can only do two things. Impersonating the joiner to the host is one online guess per attempt, and
// the host locks a peer after maxFailures and burns the code after burnFailures, so with an 8-character code from a
// 31-letter alphabet (about 2^39.6) the chance of guessing it is at most burnFailures/2^39.6. Impersonating the host
// to the joiner yields one offline test per guess against the joiner's proof; each costs pairIterations of PBKDF2 and
// the code lives five minutes, so it is bounded by that cost, not by a rate limit.
const (
	codeLength   = 8
	codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	codeTTL      = 5 * time.Minute
	sessionTTL   = time.Minute
	maxSessions  = 8
	// maxSessionsPerPeer keeps one host from holding every slot while a code is open.
	maxSessionsPerPeer = 2
	maxFailures        = 5
	burnFailures       = 15
	lockout            = 10 * time.Minute
	pairProtocolInfo   = "mortar-lan-pair-v1"

	// PairedEvent tells the window a computer finished pairing with this one.
	PairedEvent = "lan:paired"
)

// pairIterations is a var so tests can run the handshake without paying for a production-strength key.
var pairIterations = 200_000

func pairConfig() codepair.Config {
	return codepair.Config{
		Protocol:    pairProtocolInfo,
		PeerKeyInfo: "mortar-lan-peer-key",
		Iterations:  pairIterations,
		CodeLength:  codeLength,
		Alphabet:    codeAlphabet,
	}
}

// pairLibErr maps a library error to the message a player used to see: the wrapped cause, when the library added one.
func pairLibErr(err error) error {
	if inner := errors.Unwrap(err); inner != nil {
		return inner
	}
	return err
}

// PairedPeer is a computer this one trusts.
type PairedPeer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type pairedRecord struct {
	PairedPeer
	Key string `json:"key"`
}

type peersFile struct {
	Self  string         `json:"self"`
	Peers []pairedRecord `json:"peers"`
}

// peerBook is <data>/lan/peers.json: this machine's stable id and the keys of its paired computers. Without a
// directory (a service built with no data folder) it lives in memory only.
type peerBook struct {
	mu     sync.Mutex
	dir    string
	loaded bool
	file   peersFile
}

func (b *peerBook) loadLocked() error {
	if b.loaded {
		return nil
	}
	if b.dir != "" {
		if _, err := datadir.ReadJSON(filepath.Join(b.dir, "peers.json"), &b.file); err != nil {
			return fmt.Errorf("read paired computers: %w", err)
		}
	}
	if b.file.Self == "" {
		b.file.Self = randomToken()
		if err := b.saveLocked(); err != nil {
			return err
		}
	}
	b.loaded = true
	return nil
}

func (b *peerBook) saveLocked() error {
	if b.dir == "" {
		return nil
	}
	return datadir.WriteJSONIn(b.dir, "peers.json", b.file)
}

func (b *peerBook) self() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.loadLocked(); err != nil {
		return "", err
	}
	return b.file.Self, nil
}

// key returns the shared key of a paired computer, or nil.
func (b *peerBook) key(id string) []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	if id == "" || b.loadLocked() != nil {
		return nil
	}
	for _, record := range b.file.Peers {
		if record.ID == id {
			key, err := base64.RawStdEncoding.DecodeString(record.Key)
			if err != nil {
				return nil
			}
			return key
		}
	}
	return nil
}

func (b *peerBook) add(peer PairedPeer, key []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.loadLocked(); err != nil {
		return err
	}
	b.file.Peers = slices.DeleteFunc(b.file.Peers, func(r pairedRecord) bool { return r.ID == peer.ID })
	b.file.Peers = append(b.file.Peers, pairedRecord{PairedPeer: peer, Key: base64.RawStdEncoding.EncodeToString(key)})
	return b.saveLocked()
}

func (b *peerBook) remove(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.loadLocked(); err != nil {
		return err
	}
	b.file.Peers = slices.DeleteFunc(b.file.Peers, func(r pairedRecord) bool { return r.ID == id })
	return b.saveLocked()
}

func (b *peerBook) list() ([]PairedPeer, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.loadLocked(); err != nil {
		return nil, err
	}
	out := make([]PairedPeer, 0, len(b.file.Peers))
	for _, record := range b.file.Peers {
		out = append(out, record.PairedPeer)
	}
	slices.SortFunc(out, func(a, c PairedPeer) int { return strings.Compare(a.Name, c.Name) })
	return out, nil
}

type pairSession struct {
	a       []byte
	b       *ecdh.PrivateKey
	peerID  string
	name    string
	remote  string
	expires time.Time
}

type pairHost struct {
	mu       sync.Mutex
	code     string
	expires  time.Time
	sessions map[string]pairSession
	limiter  *codepair.Limiter
}

type pairBeginRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	A    string `json:"a"`
}

type pairBeginResponse struct {
	Session string `json:"session"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	B       string `json:"b"`
}

type pairFinishRequest struct {
	Session string `json:"session"`
	Proof   string `json:"proof"`
}

type pairFinishResponse struct {
	Proof string `json:"proof"`
}

// PairCode opens pairing: it returns a fresh single-use code, shown as two groups of four, that the other computer
// must enter within five minutes. A new call replaces the previous code.
func (s *Service) PairCode(ctx context.Context) (string, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return "", fmt.Errorf("turn on sharing nearby to pair a computer: %w", err)
	}
	code, err := pairConfig().NewCode()
	if err != nil {
		return "", pairLibErr(err)
	}
	s.pairing.mu.Lock()
	s.pairing.code, s.pairing.expires = code, time.Now().Add(codeTTL)
	s.pairing.sessions = map[string]pairSession{}
	s.pairing.limiter = codepair.NewLimiter(maxFailures, burnFailures, lockout)
	s.pairing.mu.Unlock()
	return code[:4] + "-" + code[4:], nil
}

func (s *Service) listening() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// Paired lists the computers this one trusts.
func (s *Service) Paired() ([]PairedPeer, error) { return s.book.list() }

// Unpair forgets a paired computer's key.
func (s *Service) Unpair(id string) error { return s.book.remove(id) }

func (s *Service) pairOpen(remote string, now time.Time) error {
	h := &s.pairing
	if h.code == "" || now.After(h.expires) {
		return errors.New("pairing is not open")
	}
	if h.limiter != nil && h.limiter.Locked(remote, now) {
		return errors.New("too many wrong codes from this computer; try again later")
	}
	return nil
}

func (s *Service) handlePairBegin(w http.ResponseWriter, r *http.Request) {
	var req pairBeginRequest
	if !readJSON(w, r, &req) {
		return
	}
	a, err := base64.RawStdEncoding.DecodeString(req.A)
	if err != nil || len(a) != 32 || req.ID == "" || len(req.ID) > 128 || len(req.Name) > 255 {
		http.Error(w, "invalid pairing request", http.StatusBadRequest)
		return
	}
	hostID, err := s.book.self()
	if err != nil {
		http.Error(w, "could not start pairing", http.StatusInternalServerError)
		return
	}
	b, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		http.Error(w, "could not start pairing", http.StatusInternalServerError)
		return
	}
	session := randomToken()
	now, remote := time.Now(), remotePeer(r)
	h := &s.pairing
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := s.pairOpen(remote, now); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	for id, sess := range h.sessions {
		if now.After(sess.expires) {
			delete(h.sessions, id)
		}
	}
	perPeer := 0
	for _, sess := range h.sessions {
		if sess.remote == remote {
			perPeer++
		}
	}
	if len(h.sessions) >= maxSessions || perPeer >= maxSessionsPerPeer {
		http.Error(w, "pairing is busy", http.StatusTooManyRequests)
		return
	}
	h.sessions[session] = pairSession{a: a, b: b, peerID: req.ID, name: cleanName(req.Name), remote: remote, expires: now.Add(sessionTTL)}
	writeJSON(w, pairBeginResponse{Session: session, ID: hostID, Name: s.deviceName(), B: base64.RawStdEncoding.EncodeToString(b.PublicKey().Bytes())})
}

func (s *Service) handlePairFinish(w http.ResponseWriter, r *http.Request) {
	var req pairFinishRequest
	if !readJSON(w, r, &req) {
		return
	}
	h := &s.pairing
	now, remote := time.Now(), remotePeer(r)
	h.mu.Lock()
	sess, ok := h.sessions[req.Session]
	delete(h.sessions, req.Session)
	err := s.pairOpen(remote, now)
	code := h.code
	if ok && err == nil {
		// The guess is counted before it is checked, so guesses made side by side cannot outnumber the limit.
		s.pairChargeLocked(remote, now)
	}
	h.mu.Unlock()
	if !ok || sess.remote != remote || now.After(sess.expires) {
		http.Error(w, "pairing session expired", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	hostID, err := s.book.self()
	if err != nil {
		http.Error(w, "could not finish pairing", http.StatusInternalServerError)
		return
	}
	a, err := ecdh.X25519().NewPublicKey(sess.a)
	if err != nil {
		http.Error(w, "invalid pairing request", http.StatusBadRequest)
		return
	}
	shared, err := sess.b.ECDH(a)
	if err != nil {
		http.Error(w, "invalid pairing request", http.StatusBadRequest)
		return
	}
	cfg := pairConfig()
	transcript := cfg.Transcript(sess.a, sess.b.PublicKey().Bytes(), sess.peerID, sess.name, hostID, cleanName(s.deviceName()))
	authKey, peerKey, err := cfg.Keys(code, shared, transcript)
	if err != nil {
		http.Error(w, "could not finish pairing", http.StatusInternalServerError)
		return
	}
	got, _ := base64.RawStdEncoding.DecodeString(req.Proof)
	if !codepair.Verify(got, codepair.Proof(authKey, "J", transcript, shared)) {
		http.Error(w, "wrong code", http.StatusForbidden)
		return
	}
	h.mu.Lock()
	if h.code != code {
		h.mu.Unlock()
		http.Error(w, "pairing is not open", http.StatusForbidden)
		return
	}
	h.code = ""
	s.pairRefundLocked(remote)
	h.mu.Unlock()
	peer := PairedPeer{ID: sess.peerID, Name: sess.name}
	if err := s.book.add(peer, peerKey); err != nil {
		http.Error(w, "could not save the pairing", http.StatusInternalServerError)
		return
	}
	if s.deps.Emit != nil {
		s.deps.Emit(PairedEvent, peer)
	}
	writeJSON(w, pairFinishResponse{Proof: base64.RawStdEncoding.EncodeToString(codepair.Proof(authKey, "H", transcript, shared))})
}

// pairChargeLocked counts one guess from remote and burns the code once the limiter says so; the caller holds h.mu.
func (s *Service) pairChargeLocked(remote string, now time.Time) {
	h := &s.pairing
	if h.limiter == nil {
		h.limiter = codepair.NewLimiter(maxFailures, burnFailures, lockout)
	}
	if h.limiter.Charge(remote, now) {
		h.code = ""
	}
}

// pairRefundLocked gives back the guess a correct code spent.
func (s *Service) pairRefundLocked(remote string) {
	if s.pairing.limiter != nil {
		s.pairing.limiter.Refund(remote, time.Now())
	}
}

// Pair enters the code shown by the computer at peerID (host:port) and stores the key both sides derive.
func (s *Service) Pair(ctx context.Context, peerID, code string) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	selfID, err := s.book.self()
	if err != nil {
		return err
	}
	a, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	aPub := a.PublicKey().Bytes()
	var begin pairBeginResponse
	if err := s.postJSON(ctx, peerID, "/pair/begin", pairBeginRequest{ID: selfID, Name: s.deviceName(), A: base64.RawStdEncoding.EncodeToString(aPub)}, &begin); err != nil {
		return err
	}
	bPub, err := base64.RawStdEncoding.DecodeString(begin.B)
	if err != nil {
		return errors.New("the other computer sent an invalid pairing reply")
	}
	b, err := ecdh.X25519().NewPublicKey(bPub)
	if err != nil {
		return errors.New("the other computer sent an invalid pairing reply")
	}
	shared, err := a.ECDH(b)
	if err != nil {
		return errors.New("the other computer sent an invalid pairing reply")
	}
	cfg := pairConfig()
	transcript := cfg.Transcript(aPub, bPub, selfID, cleanName(s.deviceName()), begin.ID, cleanName(begin.Name))
	authKey, peerKey, err := cfg.Keys(code, shared, transcript)
	if err != nil {
		return pairLibErr(err)
	}
	var finish pairFinishResponse
	proof := base64.RawStdEncoding.EncodeToString(codepair.Proof(authKey, "J", transcript, shared))
	if err := s.postJSON(ctx, peerID, "/pair/finish", pairFinishRequest{Session: begin.Session, Proof: proof}, &finish); err != nil {
		return err
	}
	got, _ := base64.RawStdEncoding.DecodeString(finish.Proof)
	if !codepair.Verify(got, codepair.Proof(authKey, "H", transcript, shared)) {
		return errors.New("the other computer did not prove it knew the code")
	}
	return s.book.add(PairedPeer{ID: begin.ID, Name: cleanName(begin.Name)}, peerKey)
}

func readJSON(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(into); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Service) postJSON(ctx context.Context, peerID, endpointPath string, body, into any) error {
	endpoint, err := shareEndpoint(peerID, endpointPath)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := postRaw(ctx, endpoint, raw)
	if err != nil {
		return fmt.Errorf("contact LAN peer: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("the other computer refused: %s", strings.TrimSpace(string(message)))
	}
	return json.NewDecoder(safefs.CapReader(resp.Body, 16<<10)).Decode(into)
}
