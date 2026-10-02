package queue

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/profile"
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
	Game     string `json:"game"`
	ModID    int    `json:"modId"`
	FileID   int    `json:"fileId"`
	Kind     string `json:"kind"`
	Repo     string `json:"repo,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Asset    string `json:"asset,omitempty"`
	Latest   bool   `json:"latest,omitempty"`
	Size     int64  `json:"size"`
	Started  int64  `json:"started"`
	Finished int64  `json:"finished"`
	Outcome  string `json:"outcome"`
	Error    string `json:"error,omitempty"`
}

func (s *Service) historyPath() string {
	return filepath.Join(s.d.Dir, historyFile)
}

func (s *Service) loadHistory() []HistoryEntry {
	b, err := os.ReadFile(s.historyPath())
	if err != nil {
		return nil
	}
	var entries []HistoryEntry
	if json.Unmarshal(b, &entries) != nil {
		return nil
	}
	return entries
}

func (s *Service) writeHistory(entries []HistoryEntry) {
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
	name := it.Name
	if name == "" {
		name = it.FileName
	}
	src := profile.KindNexus
	if it.Repo != "" {
		src = profile.KindGitHub
	}
	size := it.SizeKB << 10
	entry := HistoryEntry{
		Name: name, Version: it.Version, Source: src, Profile: it.Profile,
		Game: it.Game, ModID: it.ModID, FileID: it.FileID, Kind: it.Kind, Repo: it.Repo, Tag: it.Tag, Asset: it.Asset,
		Latest: it.Latest, Size: size, Started: started, Finished: now, Outcome: outcome, Error: it.Error,
	}
	s.pub.Lock()
	defer s.pub.Unlock()
	entries := append(s.loadHistory(), entry)
	if len(entries) > historyLimit {
		entries = entries[len(entries)-historyLimit:]
	}
	s.writeHistory(entries)
}

// History returns finished downloads, newest last, at most historyLimit.
func (s *Service) History() []HistoryEntry {
	s.pub.Lock()
	defer s.pub.Unlock()
	entries := s.loadHistory()
	if entries == nil {
		return []HistoryEntry{}
	}
	return entries
}

// ClearHistory wipes the download history file.
func (s *Service) ClearHistory() {
	s.pub.Lock()
	defer s.pub.Unlock()
	s.writeHistory(nil)
}

// RetryHistory re-enqueues the request represented by a failed or skipped history entry.
func (s *Service) RetryHistory(entry HistoryEntry) ([]Item, error) {
	return s.Add([]Request{{
		Kind: entry.Kind, Game: entry.Game, Profile: entry.Profile, ModID: entry.ModID, FileID: entry.FileID,
		Name: entry.Name, Version: entry.Version, Repo: entry.Repo, Tag: entry.Tag, Asset: entry.Asset, Latest: entry.Latest,
	}})
}
