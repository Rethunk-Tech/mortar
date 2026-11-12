package deploy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

const logFile = "deploy.log"

// syncWrites is off under go test, as in go-safefs: a test cannot observe an fsync.
var syncWrites = !testing.Testing()

// syncEvery is how many appended lines may wait for a flush to disk. A process that dies loses nothing (each line is a
// single write), so the flush guards against power loss; a folder line is flushed at once, since the folder follows.
const syncEvery = 128

// step is one line of the operation log. P is the index of a file Apply placed, U the index of one Purge took back, M
// a folder Apply is about to create.
type step struct {
	P *int   `json:"p,omitempty"`
	U *int   `json:"u,omitempty"`
	M string `json:"m,omitempty"`
	// W is the index of a file whose changed bytes were copied back into the profile; A a created file adopted into it.
	W *int   `json:"w,omitempty"`
	A string `json:"a,omitempty"`
	// R is a rescue file kept for changed bytes that had no profile to go to.
	R string `json:"r,omitempty"`
}

// opLog is the append-only record of what a deploy and its purge did, beside the first full record (deploy.json) that
// names every destination. Each completed operation is one short line, so durability is constant per file; recovery
// replays the lines over the record.
type opLog struct {
	f       *os.File
	pending int
}

func logPath(dir string) string { return filepath.Join(dir, logFile) }

func openLog(dir string) (*opLog, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	f, err := safefs.OpenFile(logPath(dir), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	return &opLog{f: f}, nil
}

// add appends one line; now flushes it to disk before returning.
func (l *opLog) add(s step, now bool) error {
	if l == nil {
		return nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if _, err := l.f.Write(append(b, '\n')); err != nil {
		return err
	}
	if l.pending++; now || l.pending >= syncEvery {
		l.pending = 0
		if !syncWrites {
			return nil
		}
		return l.f.Sync()
	}
	return nil
}

func (l *opLog) close() error {
	if l == nil {
		return nil
	}
	if !syncWrites {
		return l.f.Close()
	}
	return errors.Join(l.f.Sync(), l.f.Close())
}

// replay applies the log beside the record to m: placed files become Done, taken-back ones Undone, and the folders
// Apply made join Created. A last line a crash cut short is ignored.
func replay(m *Manifest) error {
	if m.View.JournalDir == "" {
		return nil
	}
	b, found, err := safefs.ReadFileIfExists(logPath(m.View.JournalDir))
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var s step
		if json.Unmarshal(sc.Bytes(), &s) != nil {
			continue
		}
		switch {
		case s.P != nil && *s.P >= 0 && *s.P < len(m.Ops):
			m.Ops[*s.P].Done = true
		case s.U != nil && *s.U >= 0 && *s.U < len(m.Ops):
			m.Ops[*s.U].Undone = true
		case s.R != "":
			m.notes = append(m.notes, Note{Path: s.R, Kind: NoteRescued})
		case s.M != "" && !slices.Contains(m.Created, s.M):
			m.Created = append(m.Created, s.M)
		}
	}
	return sc.Err()
}

// crashHook is set by tests to stop a deploy or purge after a named step, as a crash would.
var crashHook func(name string)

func at(name string) {
	if crashHook != nil {
		crashHook(name)
	}
}
