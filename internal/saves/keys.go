package saves

import (
	"bytes"
	"io"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/meta"
)

const modDataPrefix = "smapi/mod-data/"

var (
	opener = []byte("<key><string>")
	closer = []byte("</string>")
)

// maxKey bounds a key's length: a longer one is game data, not a mod's key, and must not make the scan buffer it.
const maxKey = 512

// distinctKeys streams r and returns each distinct <key><string>...</string></key> key once. A byte search is
// used instead of an XML decoder because saves reach hundreds of MB.
func distinctKeys(r io.Reader) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	buf := make([]byte, 1<<20)
	n := 0
	for {
		m, err := r.Read(buf[n:])
		n += m
		eof := err == io.EOF
		if err != nil && !eof {
			return nil, err
		}
		pos := 0
		for {
			i := bytes.Index(buf[pos:n], opener)
			if i < 0 {
				pos = max(pos, n-len(opener)+1)
				break
			}
			start := pos + i + len(opener)
			j := bytes.Index(buf[start:n], closer)
			if j < 0 {
				if n-start <= maxKey && !eof {
					pos += i
					break
				}
				pos = start
				continue
			}
			if j <= maxKey {
				if _, seen := out[string(buf[start:start+j])]; !seen {
					out[string(buf[start:start+j])] = struct{}{}
				}
			}
			pos = start + j + len(closer)
		}
		n = copy(buf, buf[pos:n])
		if eof {
			return out, nil
		}
	}
}

// uniqueID maps a save key to the UniqueID it belongs to: the longest prefix ending at a '/', '_' or '.' that
// index knows, ignoring case. SMAPI's own save data lives under smapi/mod-data/<uniqueid>/.
func uniqueID(key string, index map[string][]meta.Ref) (string, bool) {
	k := strings.TrimPrefix(strings.ToLower(key), modDataPrefix)
	for i := len(k) - 1; i > 0; i-- {
		if c := k[i]; c == '/' || c == '_' || c == '.' {
			if _, ok := index[k[:i]]; ok {
				return k[:i], true
			}
		}
	}
	return "", false
}
