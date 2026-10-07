package queue

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

const (
	historyFile  = "download-history.json"
	historyLimit = 1000
)

// HistoryEntry is one finished download, kept in a bounded file under the data dir.
type HistoryEntry struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Source   string `json:"source"`
	Profile  string `json:"profileId"`
	BatchID  string `json:"batchId,omitempty"`
	Game     string `json:"game"`
	ModID    int    `json:"modId"`
	FileID   int    `json:"fileId"`
	Kind     string `json:"kind"`
	Package  string `json:"package,omitempty"`
	Repo     string `json:"repo,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Asset    string `json:"asset,omitempty"`
	Latest   bool   `json:"latest,omitempty"`
	Size     int64  `json:"size"`
	Started  int64  `json:"started"`
	Finished int64  `json:"finished"`
	Outcome  string `json:"outcome"`
	Error    string `json:"error,omitempty"`
	// Override is the antivirus detection the player chose to install despite.
	Override string `json:"override,omitempty"`
}

func (s *Service) historyPath() string {
	return filepath.Join(s.d.Dir, historyFile)
}

// loadHistory returns the in-memory history, reading the file on first use; callers hold s.hist.
func (s *Service) loadHistory() []HistoryEntry {
	if !s.histLoaded {
		s.histLoaded = true
		var entries []HistoryEntry
		if _, err := datadir.ReadJSON(s.historyPath(), &entries); err == nil {
			s.history = entries
		}
	}
	return s.history
}

// writeHistory replaces the history and persists it; callers hold s.hist.
func (s *Service) writeHistory(entries []HistoryEntry) {
	s.histLoaded, s.history = true, entries
	if entries == nil {
		entries = []HistoryEntry{}
	}
	_ = datadir.WriteJSON(s.historyPath(), entries)
}

func (s *Service) recordHistory(it *Item, outcome string) {
	if it == nil {
		return
	}
	now := s.d.Now().Unix()
	started := now
	if !it.started.IsZero() {
		started = it.started.Unix()
	}
	name := cmp.Or(it.Name, it.FileName)
	src := profile.KindNexus
	switch {
	case it.Package != "":
		src = cmp.Or(it.Source, profile.KindThunderstore)
	case it.Repo != "":
		src = profile.KindGitHub
	}
	size := it.SizeKB << 10
	entry := HistoryEntry{
		Name: name, Version: it.Version, Source: src, Profile: it.Profile, BatchID: it.BatchID,
		Game: it.Game, ModID: it.ModID, FileID: it.FileID, Kind: it.Kind, Package: it.Package, Repo: it.Repo, Tag: it.Tag, Asset: it.Asset,
		Latest: it.Latest, Size: size, Started: started, Finished: now, Outcome: outcome, Error: it.Error, Override: it.ScanOverride,
	}
	s.hist.Lock()
	defer s.hist.Unlock()
	entries := append(s.loadHistory(), entry)
	if len(entries) > historyLimit {
		entries = entries[len(entries)-historyLimit:]
	}
	s.writeHistory(entries)
}

// History returns finished downloads, newest last, at most historyLimit.
func (s *Service) History() []HistoryEntry {
	s.hist.Lock()
	defer s.hist.Unlock()
	return append([]HistoryEntry{}, s.loadHistory()...)
}

// ClearHistory wipes the download history file.
func (s *Service) ClearHistory() {
	s.hist.Lock()
	defer s.hist.Unlock()
	s.writeHistory(nil)
}

// directSource is the site of a retried package that is not on Thunderstore.
func directSource(e HistoryEntry) string {
	if e.Package != "" && e.Source != profile.KindThunderstore {
		return e.Source
	}
	return ""
}

func (e HistoryEntry) request() Request {
	return Request{
		Kind: e.Kind, Game: e.Game, Profile: e.Profile, ModID: e.ModID, FileID: e.FileID,
		Name: e.Name, Version: e.Version, Package: e.Package, Source: directSource(e), Repo: e.Repo, Tag: e.Tag, Asset: e.Asset, Latest: e.Latest,
	}
}

// RetryHistory re-enqueues the request represented by a failed or skipped history entry.
func (s *Service) RetryHistory(ctx context.Context, entry HistoryEntry) ([]Item, error) {
	return s.Add(ctx, []Request{entry.request()})
}

// RetryAllResult says what RetryAllFailed did. Skipped counts failed entries left alone, by reason: "queued"
// (the download is already waiting or running), "superseded" (a newer history entry covers the same download)
// and "incomplete" (the entry lacks what a download needs).
type RetryAllResult struct {
	Requeued int            `json:"requeued"`
	Skipped  map[string]int `json:"skipped"`
}

// RetryAllFailed re-enqueues every failed history entry that is still retryable. Only the newest entry of a
// download counts, so one that later succeeded or already failed again is not queued twice.
func (s *Service) RetryAllFailed(ctx context.Context) (RetryAllResult, error) {
	res := RetryAllResult{Skipped: map[string]int{}}
	entries := s.History()
	seen := map[string]bool{}
	var reqs []Request
	s.mu.Lock()
	for _, e := range slices.Backward(entries) {
		r := e.request()
		id := fmt.Sprint(r.Game, "|", r.Profile, "|", r.ModID, "|", r.FileID, "|", r.Package, "|", r.Repo, "|", r.Tag, "|", r.Asset)
		if seen[id] {
			if e.Outcome == StateFailed {
				res.Skipped["superseded"]++
			}
			continue
		}
		seen[id] = true
		switch {
		case e.Outcome != StateFailed:
		case !r.valid():
			res.Skipped["incomplete"]++
		case slices.ContainsFunc(s.items, func(it *Item) bool { return sameDownload(it, r) }):
			res.Skipped["queued"]++
		default:
			reqs = append(reqs, r)
		}
	}
	s.mu.Unlock()
	if len(reqs) == 0 {
		return res, nil
	}
	if _, err := s.Add(ctx, reqs); err != nil {
		return res, err
	}
	res.Requeued = len(reqs)
	return res, nil
}
