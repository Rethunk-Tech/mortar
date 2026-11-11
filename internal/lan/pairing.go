package lan

import (
	"context"
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
// Construction: CPace (go-codepair) with key confirmation. The joiner sends its message, the host answers with its
// own and a confirmation tag, and the joiner answers with its tag only after the host's verifies. The long-term key
// comes from the CPace session key, so it never depends on the code alone.
//
// Threat model. A passive observer learns nothing it can test guesses against. An active attacker gets one online
// guess per run in either direction and no offline test. Posing as the joiner, its guesses are charged before the host
// answers, which locks a computer after maxFailures and closes the code after burnFailures, so with an 8-character
// code from a 31-letter alphabet (about 2^39.6) the chance is at most burnFailures/2^39.6. Posing as the host, it gets
// one guess per run the user starts by typing the code, which lives five minutes.
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

func pairConfig() codepair.Config {
	return codepair.Config{
		Protocol:    pairProtocolInfo,
		PeerKeyInfo: "mortar-lan-peer-key",
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

// pairSession is one run the host answered; code is the code it was answered under, so a run outlives no new code.
type pairSession struct {
	host    *codepair.Host
	code    string
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
	Msg string `json:"msg"`
}

type pairBeginResponse struct {
	Session string `json:"session"`
	Reply   string `json:"reply"`
}

type pairFinishRequest struct {
	Session string `json:"session"`
	Confirm string `json:"confirm"`
}

// validPeer keeps the id and name limits the peer book has always had for what a pairing carried.
func validPeer(p codepair.Pairing) bool {
	return p.PeerID != "" && len(p.PeerID) <= 128 && len(p.PeerName) <= 255
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
	limiter, err := codepair.NewLimiter(maxFailures, burnFailures, lockout)
	if err != nil {
		return "", err
	}
	s.pairing.mu.Lock()
	s.pairing.code, s.pairing.expires = code, time.Now().Add(codeTTL)
	s.pairing.sessions = map[string]pairSession{}
	s.pairing.limiter = limiter
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

func (s *Service) pairOpen(now time.Time) error {
	h := &s.pairing
	if h.code == "" || now.After(h.expires) {
		return errors.New("pairing is not open")
	}
	return nil
}

func (s *Service) handlePairBegin(w http.ResponseWriter, r *http.Request) {
	var req pairBeginRequest
	if !readJSON(w, r, &req) {
		return
	}
	msg, err := base64.RawStdEncoding.DecodeString(req.Msg)
	if err != nil {
		http.Error(w, "invalid pairing request", http.StatusBadRequest)
		return
	}
	hostID, err := s.book.self()
	if err != nil {
		http.Error(w, "could not start pairing", http.StatusInternalServerError)
		return
	}
	session := randomToken()
	now, remote := time.Now(), remotePeer(r)
	h := &s.pairing
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := s.pairOpen(now); err != nil {
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
	// The reply lets the sender test one guess at the code, so the guess is counted before it is answered.
	if err := s.pairChargeLocked(remote, now); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	host, err := pairConfig().NewHost(h.code, hostID, cleanName(s.deviceName()))
	if err != nil {
		http.Error(w, "could not start pairing", http.StatusInternalServerError)
		return
	}
	reply, err := host.Accept(msg)
	if err != nil {
		// A message the host refuses tested no guess.
		s.pairRefundLocked(remote)
		http.Error(w, "invalid pairing request", http.StatusBadRequest)
		return
	}
	h.sessions[session] = pairSession{host: host, code: h.code, remote: remote, expires: now.Add(sessionTTL)}
	writeJSON(w, pairBeginResponse{Session: session, Reply: base64.RawStdEncoding.EncodeToString(reply)})
}

func (s *Service) handlePairFinish(w http.ResponseWriter, r *http.Request) {
	var req pairFinishRequest
	if !readJSON(w, r, &req) {
		return
	}
	confirm, err := base64.RawStdEncoding.DecodeString(req.Confirm)
	if err != nil {
		http.Error(w, "invalid pairing request", http.StatusBadRequest)
		return
	}
	h := &s.pairing
	now, remote := time.Now(), remotePeer(r)
	h.mu.Lock()
	sess, ok := h.sessions[req.Session]
	delete(h.sessions, req.Session)
	h.mu.Unlock()
	if !ok || sess.remote != remote || now.After(sess.expires) {
		http.Error(w, "pairing session expired", http.StatusForbidden)
		return
	}
	p, err := sess.host.Finish(confirm)
	if err != nil {
		http.Error(w, "wrong code", http.StatusForbidden)
		return
	}
	if !validPeer(p) {
		http.Error(w, "invalid pairing request", http.StatusBadRequest)
		return
	}
	h.mu.Lock()
	if h.code != sess.code || s.pairOpen(now) != nil {
		h.mu.Unlock()
		http.Error(w, "pairing is not open", http.StatusForbidden)
		return
	}
	h.code = ""
	s.pairRefundLocked(remote)
	h.mu.Unlock()
	peer := PairedPeer{ID: p.PeerID, Name: cleanName(p.PeerName)}
	if err := s.book.add(peer, p.PeerKey); err != nil {
		http.Error(w, "could not save the pairing", http.StatusInternalServerError)
		return
	}
	if s.deps.Emit != nil {
		s.deps.Emit(PairedEvent, peer)
	}
	w.WriteHeader(http.StatusNoContent)
}

// pairChargeLocked counts one guess from remote, or refuses it when remote is locked out or the code's budget is
// spent, which also closes the code. The caller holds h.mu.
func (s *Service) pairChargeLocked(remote string, now time.Time) error {
	h := &s.pairing
	locked, burned := h.limiter.Charge(remote, now)
	if burned {
		h.code = ""
		return errors.New("pairing is not open")
	}
	if locked {
		return errors.New("too many wrong codes from this computer; try again later")
	}
	return nil
}

// pairRefundLocked gives back the guess a correct code spent.
func (s *Service) pairRefundLocked(remote string) {
	s.pairing.limiter.Refund(remote, time.Now())
}

// Pair enters the code shown by the computer at peerID (host:port) and stores the key both sides derive. The joiner
// stores it only after the host accepted its confirmation, so a refused run leaves neither side paired.
func (s *Service) Pair(ctx context.Context, peerID, code string) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	selfID, err := s.book.self()
	if err != nil {
		return err
	}
	joiner, msg, err := pairConfig().NewJoiner(code, selfID, cleanName(s.deviceName()))
	if err != nil {
		return pairLibErr(err)
	}
	var begin pairBeginResponse
	if err := s.postJSON(ctx, peerID, "/pair/begin", pairBeginRequest{Msg: base64.RawStdEncoding.EncodeToString(msg)}, &begin); err != nil {
		return err
	}
	reply, err := base64.RawStdEncoding.DecodeString(begin.Reply)
	if err != nil {
		return errors.New("the other computer sent an invalid pairing reply")
	}
	confirm, p, err := joiner.Finish(reply)
	if errors.Is(err, codepair.ErrConfirmation) {
		return errors.New("the other computer did not prove it knew the code")
	}
	if err != nil || !validPeer(p) {
		return errors.New("the other computer sent an invalid pairing reply")
	}
	if err := s.postJSON(ctx, peerID, "/pair/finish", pairFinishRequest{Session: begin.Session, Confirm: base64.RawStdEncoding.EncodeToString(confirm)}, nil); err != nil {
		return err
	}
	return s.book.add(PairedPeer{ID: p.PeerID, Name: cleanName(p.PeerName)}, p.PeerKey)
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
	if into == nil {
		return nil
	}
	return json.NewDecoder(safefs.CapReader(resp.Body, 16<<10)).Decode(into)
}
