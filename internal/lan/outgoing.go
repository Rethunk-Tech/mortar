package lan

import (
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	// OutgoingEvent carries a sent profile's file-transfer progress to the sender's window.
	OutgoingEvent = "lan:outgoing"

	OutgoingSending   = "sending"
	OutgoingDone      = "done"
	OutgoingFailed    = "failed"
	OutgoingCancelled = "cancelled"

	// outgoingStall is how long a pull may go quiet before the sender gives up on the other computer.
	outgoingStall = 2 * time.Minute
	// outgoingKept is how many finished transfers a reopened window can still list.
	outgoingKept = 20
	maxReasonLen = 200
)

// OutgoingTransfer is one paired computer pulling a sent profile's files from this one.
type OutgoingTransfer struct {
	ID         int    `json:"id"`
	Peer       string `json:"peer"`
	Profile    string `json:"profile"`
	Current    int    `json:"current"`
	Total      int    `json:"total"`
	Bytes      int64  `json:"bytes"`
	TotalBytes int64  `json:"totalBytes"`
	State      string `json:"state"`
	Reason     string `json:"reason,omitempty"`
}

type outgoingState struct {
	OutgoingTransfer
	game    string
	started time.Time
	emitted time.Time
	stall   *time.Timer
}

// outgoingReport is what the pulling computer tells the sender: the files it still needs, then how it ended.
type outgoingReport struct {
	State  string   `json:"state"`
	Keys   []string `json:"keys,omitempty"`
	Reason string   `json:"reason,omitempty"`
}

const outgoingPlan = "plan"

func terminal(state string) bool { return state != OutgoingSending }

// OutgoingTransfers lists the sent profiles' transfers, so a reopened window catches up.
func (s *Service) OutgoingTransfers() []OutgoingTransfer {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	out := make([]OutgoingTransfer, 0, len(s.outgoing))
	for _, o := range s.outgoing {
		out = append(out, o.OutgoingTransfer)
	}
	slices.SortFunc(out, func(a, b OutgoingTransfer) int { return a.ID - b.ID })
	return out
}

func (s *Service) startOutgoing(token, game, peer, profile string, total int) {
	s.outMu.Lock()
	s.nextOutgoing++
	o := &outgoingState{game: game}
	o.ID, o.Peer, o.Profile, o.Total, o.State = s.nextOutgoing, peer, profile, total, OutgoingSending
	o.stall = time.AfterFunc(outgoingStall, func() {
		s.updateOutgoing(token, func(o *OutgoingTransfer) {
			o.State, o.Reason = OutgoingFailed, "no answer from the other computer"
		})
	})
	s.outgoing[token] = o
	for key, old := range s.outgoing {
		if terminal(old.State) && old.ID <= s.nextOutgoing-outgoingKept {
			delete(s.outgoing, key)
		}
	}
	o.started, o.emitted = time.Now(), time.Now()
	snapshot := o.OutgoingTransfer
	s.outMu.Unlock()
	s.emitOutgoing(snapshot)
}

// updateOutgoing applies change to a running transfer; a finished one never changes again. Progress events are
// throttled, a state change is always sent.
func (s *Service) updateOutgoing(token string, change func(*OutgoingTransfer)) {
	s.outMu.Lock()
	o, ok := s.outgoing[token]
	if !ok || terminal(o.State) {
		s.outMu.Unlock()
		return
	}
	change(&o.OutgoingTransfer)
	if o.TotalBytes > 0 {
		o.Bytes = min(o.Bytes, o.TotalBytes)
	}
	finished := terminal(o.State)
	if finished {
		o.stall.Stop()
	} else {
		o.stall.Reset(outgoingStall)
	}
	now := time.Now()
	if !finished && now.Sub(o.emitted) < progressEvery {
		s.outMu.Unlock()
		return
	}
	o.emitted = now
	snapshot := o.OutgoingTransfer
	started := o.started
	s.outMu.Unlock()
	if finished {
		logOutgoing(snapshot, time.Since(started))
	}
	s.emitOutgoing(snapshot)
}

func logOutgoing(o OutgoingTransfer, took time.Duration) {
	took = took.Round(time.Millisecond)
	switch o.State {
	case OutgoingDone:
		log.Printf("lan: sent %q files to %s: %d files, %d bytes in %s", o.Profile, o.Peer, o.Total, o.Bytes, took)
	case OutgoingCancelled:
		log.Printf("lan: %s cancelled copying %q after %d of %d files", o.Peer, o.Profile, o.Current, o.Total)
	default:
		log.Printf("lan: copying %q to %s failed after %d of %d files: %s", o.Profile, o.Peer, o.Current, o.Total, o.Reason)
	}
}

func (s *Service) emitOutgoing(o OutgoingTransfer) {
	if s.deps.Emit != nil {
		s.deps.Emit(OutgoingEvent, o)
	}
}

// handleOutgoing takes the pulling computer's plan and ending; the transfer token is the credential.
func (s *Service) handleOutgoing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	s.outMu.Lock()
	o, known := s.outgoing[token]
	var game, peer string
	if known {
		game, peer = o.game, o.Peer
	}
	s.outMu.Unlock()
	if token == "" || !known {
		http.Error(w, "store transfer is not authorized", http.StatusForbidden)
		return
	}
	var report outgoingReport
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&report); err != nil {
		http.Error(w, "invalid report", http.StatusBadRequest)
		return
	}
	reason := report.Reason[:min(len(report.Reason), maxReasonLen)]
	switch report.State {
	case outgoingPlan:
		wanted := 0
		var bytes int64
		for _, key := range report.Keys {
			if !s.grantAllows(token, game, key) {
				continue
			}
			wanted++
			bytes += s.entrySize(game, key)
		}
		log.Printf("lan: %s pulls %d of %d files, %d bytes", peer, wanted, len(report.Keys), bytes)
		s.updateOutgoing(token, func(o *OutgoingTransfer) {
			o.Total, o.TotalBytes = wanted, bytes
			if wanted == 0 {
				o.State = OutgoingDone
			}
		})
	case OutgoingDone:
		s.updateOutgoing(token, func(o *OutgoingTransfer) {
			o.Current, o.Bytes, o.State = o.Total, o.TotalBytes, OutgoingDone
		})
		s.revokeGrant(token)
	case OutgoingCancelled:
		s.updateOutgoing(token, func(o *OutgoingTransfer) { o.State, o.Reason = report.State, reason })
		s.revokeGrant(token)
	case OutgoingFailed:
		s.updateOutgoing(token, func(o *OutgoingTransfer) { o.State, o.Reason = report.State, reason })
	default:
		http.Error(w, "unknown state", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) entrySize(game, key string) int64 {
	root, err := s.deps.Store.Path(game, key)
	if err != nil {
		return 0
	}
	var total int64
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

// countingWriter reports the bytes a served store entry has written.
// Each write gets its own deadline, so only a puller that stops reading for idle is dropped.
type countingWriter struct {
	http.ResponseWriter
	add  func(int64)
	idle time.Duration
}

func (w countingWriter) Write(p []byte) (int, error) {
	_ = http.NewResponseController(w.ResponseWriter).SetWriteDeadline(time.Now().Add(w.idle))
	n, err := w.ResponseWriter.Write(p)
	w.add(int64(n))
	return n, err
}
