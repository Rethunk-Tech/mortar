package sharehost

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/share"
)

const (
	codeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	maxCodeTries = 16
)

type handler struct {
	store Store
	now   func() time.Time
	limit *ipLimiter
	mux   http.Handler
	// trustProxyIP lets the Do-Connecting-Ip header name the client. Only safe behind a proxy that overwrites it.
	trustProxyIP bool
	locks        [64]sync.Mutex
}

// Option configures the handler.
type Option func(*handler)

// WithTrustedProxyIP takes the client address for rate limiting from the Do-Connecting-Ip header, which DigitalOcean
// App Platform sets and overwrites on every request. Without a proxy that does so, a client can forge it.
func WithTrustedProxyIP() Option { return func(h *handler) { h.trustProxyIP = true } }

func New(store Store, opts ...Option) *handler {
	h := &handler{
		store: store,
		now:   func() time.Time { return time.Now().UTC() },
		limit: newIPLimiter(createsPerHour, time.Hour),
	}
	for _, o := range opts {
		o(h)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/shares", h.create)
	mux.HandleFunc("GET /api/shares/ready", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ready": true})
	})
	mux.HandleFunc("PUT /api/shares/{code}", h.update)
	mux.HandleFunc("GET /api/shares/{code}", h.getLatest)
	mux.HandleFunc("GET /api/shares/{code}/{n}", h.getVersion)
	h.mux = mux
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type payloadBody struct {
	Payload string `json:"payload"`
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	body, ok := readPayload(w, r)
	if !ok {
		return
	}
	if !h.limit.allow(h.clientIP(r), h.now()) {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	shared, err := share.Parse(body.Payload)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	token, err := randomToken()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	now := h.now()
	meta := Meta{TokenHash: hashToken(token), Latest: 1, Created: now}
	ver := Version{Payload: body.Payload, Game: shared.Game}
	var code string
	for range maxCodeTries {
		code, err = randomCode()
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		err = h.store.Create(r.Context(), code, meta, ver)
		if err == nil {
			writeJSON(w, http.StatusCreated, map[string]any{"code": code, "token": token, "version": 1})
			return
		}
		if !errors.Is(err, ErrExists) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// lock serialises writers of one code's meta so a read's LastRead write cannot roll back a newer Latest.
func (h *handler) lock(code string) func() {
	sum := sha256.Sum256([]byte(code))
	mu := &h.locks[int(sum[0])%len(h.locks)]
	mu.Lock()
	return mu.Unlock
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if !validCode(code) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer h.lock(code)()
	meta, err := h.store.Meta(r.Context(), code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if !tokenOK(bearer(r), meta.TokenHash) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	body, ok := readPayload(w, r)
	if !ok {
		return
	}
	shared, err := share.Parse(body.Payload)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	n := meta.Latest + 1
	if err := h.store.SaveVersion(r.Context(), code, n, Version{Payload: body.Payload, Game: shared.Game}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	meta.Latest = n
	if err := h.store.SaveMeta(r.Context(), code, meta); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"version": n})
}

func (h *handler) getLatest(w http.ResponseWriter, r *http.Request) {
	h.get(w, r, 0)
}

func (h *handler) getVersion(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n < 1 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	h.get(w, r, n)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request, n int) {
	code := r.PathValue("code")
	if !validCode(code) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	meta, err := h.store.Meta(r.Context(), code)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if n == 0 {
		n = meta.Latest
	}
	ver, err := h.store.LoadVersion(r.Context(), code, n)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	now := h.now()
	if meta.LastRead.IsZero() || now.Sub(meta.LastRead) >= lastReadGap {
		h.touch(r.Context(), code, now)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"code":    code,
		"version": n,
		"game":    ver.Game,
		"payload": ver.Payload,
	})
}

// touch records a read on the current meta, re-read under the lock.
func (h *handler) touch(ctx context.Context, code string, now time.Time) {
	defer h.lock(code)()
	meta, err := h.store.Meta(ctx, code)
	if err != nil {
		return
	}
	meta.LastRead = now
	_ = h.store.SaveMeta(ctx, code, meta)
}

func readPayload(w http.ResponseWriter, r *http.Request) (payloadBody, bool) {
	var body payloadBody
	r.Body = http.MaxBytesReader(w, r.Body, maxPayload+1024)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&body); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) || errors.Is(err, io.ErrUnexpectedEOF) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return payloadBody{}, false
		}
		w.WriteHeader(http.StatusBadRequest)
		return payloadBody{}, false
	}
	if len(body.Payload) > maxPayload {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return payloadBody{}, false
	}
	return body, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *handler) clientIP(r *http.Request) string {
	if h.trustProxyIP {
		if ip := strings.TrimSpace(r.Header.Get("Do-Connecting-Ip")); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func bearer(r *http.Request) string {
	const prefix = "Bearer "
	a := r.Header.Get("Authorization")
	if !strings.HasPrefix(a, prefix) {
		return ""
	}
	return strings.TrimSpace(a[len(prefix):])
}

func tokenOK(token, wantHash string) bool {
	got := hashToken(token)
	if len(got) != len(wantHash) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(wantHash)) == 1
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func validCode(code string) bool {
	if len(code) != codeLen {
		return false
	}
	for i := 0; i < len(code); i++ {
		c := code[i]
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func randomCode() (string, error) {
	return randomAlpha(codeLen)
}

func randomToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func randomAlpha(n int) (string, error) {
	out := make([]byte, n)
	buf := make([]byte, n)
	i := 0
	for i < n {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, c := range buf {
			if int(c) >= 256-256%len(codeAlphabet) {
				continue
			}
			out[i] = codeAlphabet[int(c)%len(codeAlphabet)]
			i++
			if i == n {
				break
			}
		}
	}
	return string(out), nil
}

type ipLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	hits   map[string][]time.Time
}

func newIPLimiter(n int, window time.Duration) *ipLimiter {
	return &ipLimiter{max: n, window: window, hits: map[string][]time.Time{}}
}

func (l *ipLimiter) allow(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-l.window)
	q := l.hits[ip]
	i := 0
	for i < len(q) && !q[i].After(cutoff) {
		i++
	}
	q = q[i:]
	if len(q) == 0 {
		delete(l.hits, ip)
	}
	if len(q) >= l.max {
		l.hits[ip] = q
		return false
	}
	l.hits[ip] = append(q, now)
	return true
}

// RunSweep ticks every `every` (typically 24h) and logs deleted/kept counts, including zero.
func RunSweep(ctx context.Context, store Store, every time.Duration, now func() time.Time) {
	if now == nil {
		now = time.Now
	}
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		cutoff := now().Add(-retainFor)
		deleted, kept, err := store.Sweep(ctx, cutoff)
		if err != nil {
			log.Printf("sharehost sweep: deleted %d, kept %d: %v", deleted, kept, err)
		} else {
			log.Printf("sharehost sweep: deleted %d, kept %d", deleted, kept)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
