package lan

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Everything a paired computer sends after pairing (the share, the transfer token, the store files and the pull
// reports) travels sealed with AES-GCM under keys only the two computers can derive, so a host that relays the
// handshake, or one that listens on the network, gets nothing usable. The session key comes from the pair key and
// both sides' fresh handshake values, the dialed address and both ids; a transfer key follows from it and the token.
const (
	sealHeader   = "X-Mortar-Seal"
	tagHeader    = "X-Mortar-Tag"
	handleHeader = "X-Mortar-Handle"
	nonceHeader  = "X-Mortar-Nonce"
	sealedReply  = "X-Mortar-Sealed"
	saltLen      = 16
	frameMax     = 64 << 10
	saltTTL      = 10 * time.Minute
	maxSalts     = 8192
)

var errUnsealed = errors.New("the reply was not sealed with the pairing key")

// kdf derives a 32-byte key from secret for one purpose; parts are length-prefixed so they cannot run together.
func kdf(secret []byte, purpose string, parts ...string) []byte {
	h := sha256.New()
	for _, p := range parts {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(p))) //nolint:gosec // G115: a kdf part is a short literal
		h.Write(n[:])
		h.Write([]byte(p))
	}
	key, err := hkdf.Key(sha256.New, secret, h.Sum(nil), "mortar-lan-"+purpose, 32)
	if err != nil {
		panic(err) // only for a length hkdf refuses; 32 is fine
	}
	return key
}

// sessionKey is the key of one handshake between caller and answerer; host is the address the caller dialed.
func sessionKey(pairKey []byte, challenge, nonce, callerID, answererID, host string) []byte {
	return kdf(pairKey, "session-v6", challenge, nonce, callerID, answererID, host)
}

// transferKey is the key of one accepted share's file pulls and reports.
func transferKey(session []byte, token string) []byte { return kdf(session, "transfer", token) }

// tokenHandle names a transfer token on the wire without revealing it.
func tokenHandle(token string) string {
	sum := sha256.Sum256([]byte("mortar-lan-handle|" + token))
	return hex.EncodeToString(sum[:16])
}

func newAEAD(key []byte) cipher.AEAD {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err) // key is always 32 bytes
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	return aead
}

func frameNonce(counter uint64) []byte {
	nonce := make([]byte, 12)
	binary.BigEndian.PutUint64(nonce[4:], counter)
	return nonce
}

// sealWriter writes plaintext as authenticated frames: flag, length, ciphertext. The last frame carries flag 1, so a
// stream cut short is told from one that ended.
type sealWriter struct {
	w    io.Writer
	aead cipher.AEAD
	n    uint64
	buf  []byte
	done bool
}

func newSealWriter(w io.Writer, key []byte) *sealWriter {
	return &sealWriter{w: w, aead: newAEAD(key)}
}

func (s *sealWriter) frame(final bool) error {
	flag := byte(0)
	if final {
		flag = 1
	}
	ct := s.aead.Seal(nil, frameNonce(s.n), s.buf, []byte{flag})
	s.n++
	head := make([]byte, 5)
	head[0] = flag
	binary.BigEndian.PutUint32(head[1:], uint32(len(ct))) //nolint:gosec // G115: a frame holds at most frameMax plus the AEAD overhead
	s.buf = s.buf[:0]
	if _, err := s.w.Write(head); err != nil {
		return err
	}
	_, err := s.w.Write(ct)
	return err
}

func (s *sealWriter) Write(p []byte) (int, error) {
	if s.done {
		return 0, io.ErrClosedPipe
	}
	total := len(p)
	for len(p) > 0 {
		room := min(frameMax-len(s.buf), len(p))
		s.buf = append(s.buf, p[:room]...)
		p = p[room:]
		if len(s.buf) == frameMax {
			if err := s.frame(false); err != nil {
				return total - len(p), err
			}
		}
	}
	return total, nil
}

// Close writes the final frame.
func (s *sealWriter) Close() error {
	if s.done {
		return nil
	}
	s.done = true
	return s.frame(true)
}

type sealReader struct {
	r    io.Reader
	aead cipher.AEAD
	n    uint64
	buf  []byte
	last bool
	err  error
}

func newSealReader(r io.Reader, key []byte) *sealReader {
	return &sealReader{r: r, aead: newAEAD(key)}
}

func (s *sealReader) Read(p []byte) (int, error) {
	for len(s.buf) == 0 {
		if s.err != nil {
			return 0, s.err
		}
		if s.last {
			return 0, io.EOF
		}
		s.err = s.next()
	}
	n := copy(p, s.buf)
	s.buf = s.buf[n:]
	return n, nil
}

func (s *sealReader) next() error {
	head := make([]byte, 5)
	if _, err := io.ReadFull(s.r, head); err != nil {
		if errors.Is(err, io.EOF) {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	size := binary.BigEndian.Uint32(head[1:])
	if head[0] > 1 || size > frameMax+uint32(s.aead.Overhead()) { //nolint:gosec // G115: Overhead is a small positive constant
		return errors.New("sealed stream is malformed")
	}
	ct := make([]byte, size)
	if _, err := io.ReadFull(s.r, ct); err != nil {
		return io.ErrUnexpectedEOF
	}
	pt, err := s.aead.Open(nil, frameNonce(s.n), ct, head[:1])
	if err != nil {
		return errors.New("sealed stream failed authentication")
	}
	s.n++
	s.buf, s.last = pt, head[0] == 1
	return nil
}

func tagFor(key, salt []byte, method, path string) string {
	aead := newAEAD(kdf(key, "tag", string(salt)))
	return base64.RawStdEncoding.EncodeToString(aead.Seal(frameNonce(0), frameNonce(0), nil, []byte(method+"|"+path)))
}

func tagValid(key, salt []byte, method, path, tag string) bool {
	raw, err := base64.RawStdEncoding.DecodeString(tag)
	aead := newAEAD(kdf(key, "tag", string(salt)))
	if err != nil || len(raw) != 12+aead.Overhead() {
		return false
	}
	_, err = aead.Open(nil, raw[:12], raw[12:], []byte(method+"|"+path))
	return err == nil
}

type sealedCtxKey struct{}

// sealedPeer is the paired computer a sealed /share came from and the session key it was sealed with.
type sealedPeer struct {
	id      string
	session []byte
}

// resolveFunc finds the key a sealed request is under, the transfer token it names (if any) and who sent it.
type resolveFunc func(r *http.Request) (key []byte, token string, peer *sealedPeer, err error)

// seenSalt records a salt, refusing one already used: a captured request replayed later is not served twice.
func (s *Service) seenSalt(salt string) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.salts == nil {
		s.salts = map[string]time.Time{}
	}
	if _, dup := s.salts[salt]; dup {
		return true
	}
	if len(s.salts) >= maxSalts {
		for k, at := range s.salts {
			if now.After(at) {
				delete(s.salts, k)
			}
		}
		if len(s.salts) >= maxSalts {
			return true
		}
	}
	s.salts[salt] = now.Add(saltTTL)
	return false
}

// sealed opens a request that carries the seal header under the key resolve names, and seals the handler's reply
// the same way. A request without the header goes straight through when optional (an unpaired share) and is
// refused otherwise.
func (s *Service) sealed(resolve resolveFunc, optional bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		saltText := r.Header.Get(sealHeader)
		if saltText == "" {
			if optional {
				next(w, r)
				return
			}
			http.Error(w, "this request must be sealed", http.StatusForbidden)
			return
		}
		salt, err := base64.RawStdEncoding.DecodeString(saltText)
		if err != nil || len(salt) != saltLen {
			http.Error(w, "invalid seal", http.StatusBadRequest)
			return
		}
		key, token, peer, err := resolve(r)
		if err != nil || !tagValid(key, salt, r.Method, r.URL.Path, r.Header.Get(tagHeader)) {
			http.Error(w, "not authorized", http.StatusForbidden)
			return
		}
		if s.seenSalt(saltText) {
			http.Error(w, "request was already used", http.StatusForbidden)
			return
		}
		if r.ContentLength != 0 && r.Body != nil && r.Body != http.NoBody {
			r.Body = struct {
				io.Reader
				io.Closer
			}{newSealReader(r.Body, kdf(key, "request", string(salt))), r.Body}
		}
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if peer != nil {
			r = r.WithContext(context.WithValue(r.Context(), sealedCtxKey{}, peer))
		}
		w.Header().Set(sealedReply, "1")
		sw := &sealedWriter{ResponseWriter: w, seal: newSealWriter(w, kdf(key, "reply", string(salt)))}
		defer func() { _ = sw.seal.Close() }()
		next(sw, r)
	}
}

type sealedWriter struct {
	http.ResponseWriter
	seal *sealWriter
}

func (w *sealedWriter) Write(p []byte) (int, error) { return w.seal.Write(p) }

// FlushError sends what is buffered, so a handler that flushes still reaches the reader promptly.
func (w *sealedWriter) FlushError() error {
	if len(w.seal.buf) > 0 {
		if err := w.seal.frame(false); err != nil {
			return err
		}
	}
	return http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *sealedWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// sealedDo sends a request sealed under key; handle names the transfer token for store and report calls. A reply
// that is not sealed is an error, never an answer.
func sealedDo(ctx context.Context, method, endpoint string, key []byte, handle string, body io.Reader, contentType string, headers ...string) (*http.Response, error) {
	salt := make([]byte, saltLen)
	_, _ = rand.Read(salt)
	var reader io.Reader
	if body != nil {
		pr, pw := io.Pipe()
		go func() {
			sw := newSealWriter(pw, kdf(key, "request", string(salt)))
			_, err := io.Copy(sw, body)
			if err == nil {
				err = sw.Close()
			}
			_ = pw.CloseWithError(err)
		}()
		reader = pr
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set(sealHeader, base64.RawStdEncoding.EncodeToString(salt))
	req.Header.Set(tagHeader, tagFor(key, salt, method, req.URL.Path))
	if handle != "" {
		req.Header.Set(handleHeader, handle)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := peerClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Header.Get(sealedReply) != "1" {
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("contact LAN peer: %w", errUnsealed)
		}
		return resp, nil
	}
	resp.Body = struct {
		io.Reader
		io.Closer
	}{newSealReader(resp.Body, kdf(key, "reply", string(salt))), resp.Body}
	return resp, nil
}
