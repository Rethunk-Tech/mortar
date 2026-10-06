package saves

import (
	"bytes"
	"io"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/meta"
)

const modDataPrefix = "smapi/mod-data/"

var (
	opener = []byte("<key><string>")
	closer = []byte("</string>")
)

// maxKey bounds a key's length: a longer one is game data, not a mod's key, and must not make the scan buffer it.
const maxKey = 512

// firstTag finds the text of the first <name>…</name> in a stream read chunk by chunk.
type firstTag struct {
	open, close []byte
	max         int
	value       string
	found       bool
}

func newFirstTag(name string, maxLen int) *firstTag {
	return &firstTag{open: []byte("<" + name + ">"), close: []byte("</" + name + ">"), max: maxLen}
}

// scan looks in buf[:n] and returns the lowest offset the next chunk must keep, at most pos.
func (t *firstTag) scan(buf []byte, n, pos int, eof bool) int {
	if t.found {
		return pos
	}
	i := bytes.Index(buf[:n], t.open)
	if i < 0 {
		return min(pos, max(0, n-len(t.open)+1))
	}
	start := i + len(t.open)
	j := bytes.Index(buf[start:n], t.close)
	switch {
	case j < 0 && n-start <= t.max && !eof:
		return min(pos, i)
	case j >= 0 && j <= t.max:
		t.value, t.found = strings.TrimSpace(string(buf[start:start+j])), true
	}
	return pos
}

// saveFarm is a save's Game1.whichFarm and, for a custom farm, the Data/AdditionalFarms id in whichModFarm.
type saveFarm struct {
	which int
	has   bool
	mod   string
}

// distinctKeys streams r and returns each distinct <key><string>...</string></key> key once. A byte search is
// used instead of an XML decoder because saves reach hundreds of MB. The farm tags are the first <whichFarm>
// and <whichModFarm>, found in the same pass.
func distinctKeys(r io.Reader) (map[string]struct{}, saveFarm, error) {
	out := map[string]struct{}{}
	buf := make([]byte, 1<<20)
	n := 0
	which, modFarm := newFirstTag("whichFarm", 16), newFirstTag("whichModFarm", maxKey)
	for {
		m, err := r.Read(buf[n:])
		n += m
		eof := err == io.EOF
		if err != nil && !eof {
			return nil, saveFarm{}, err
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
		pos = which.scan(buf, n, pos, eof)
		pos = modFarm.scan(buf, n, pos, eof)
		n = copy(buf, buf[pos:n])
		if eof {
			farm := saveFarm{has: which.found}
			farm.which, _ = strconv.Atoi(which.value)
			// A custom farm is written as its id or as a ModFarmType element holding <Id>.
			farm.mod = modFarm.value
			id := newFirstTag("Id", maxKey)
			id.scan([]byte(farm.mod), len(farm.mod), 0, true)
			if id.found {
				farm.mod = id.value
			}
			return out, farm, nil
		}
	}
}

// uniqueID maps a save key to the mod id it belongs to: the longest prefix ending at a '/', '_' or '.' that
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
