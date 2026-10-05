package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/store"
)

// Health finding kinds.
const (
	HealthMissing  = "missing"
	HealthDrift    = "drift"
	HealthUnused   = "unused"
	HealthSnapshot = "snapshot"
	HealthJournal  = "journal"
)

// Repair action ids. Download and cleanup run in the app, which owns the download queue and the storage cleanup
// dialog; RepairProfile applies the others.
const (
	RepairDownload     = "download"
	RepairRestore      = "restore"
	RepairRevert       = "revert"
	RepairCleanup      = "cleanup"
	RepairDropSnapshot = "drop-snapshot"
	RepairRecover      = "recover"
)

// HealthFinding is one problem ProfileHealth found.
type HealthFinding struct {
	// ID is stable across checks, so a repair names what the user saw.
	ID   string `json:"id"`
	Kind string `json:"kind"`
	// Cause narrows the kind: deleted or changed for drift, unreadable or configs for a history snapshot.
	Cause string `json:"cause,omitempty"`
	// Items name what the finding is about: mods, store items, journal folders, or the history event's label.
	Items []string `json:"items"`
	// At is the history event's time for a snapshot finding.
	At time.Time `json:"at,omitzero"`
	// Repair is the action id that fixes it, "" when Mortar cannot.
	Repair string `json:"repair"`
	// Entries are the profile entries a download repair fetches again.
	Entries []Entry `json:"entries,omitempty"`
}

// ProfileHealth checks a profile's store items, mods folder, history and launch journals.
func (s *Service) ProfileHealth(game, id string) ([]HealthFinding, error) {
	p, err := s.store.read(game, id)
	if err != nil {
		return nil, err
	}
	out, err := s.store.missingFindings(game, p)
	if err != nil {
		return nil, err
	}
	drift, err := s.store.ScanModsDrift(game, id)
	if err != nil {
		return nil, err
	}
	out = append(out, driftFindings(p, drift)...)
	if s.HealthKeep != nil && s.store.items != nil {
		keep, err := s.HealthKeep()
		if err != nil {
			return nil, err
		}
		report, err := s.store.items.Report(keep)
		if err != nil {
			return nil, err
		}
		if f, ok := unusedFinding(game, report[game].Unused); ok {
			out = append(out, f)
		}
	}
	snaps, err := s.store.snapshotFindings(game, id)
	if err != nil {
		return nil, err
	}
	out = append(out, snaps...)
	if s.HealthJournals != nil {
		if dirs := s.HealthJournals(game); len(dirs) > 0 {
			out = append(out, HealthFinding{
				ID: HealthJournal + ":" + game, Kind: HealthJournal, Items: dirs, Repair: RepairRecover,
			})
		}
	}
	if out == nil {
		out = []HealthFinding{}
	}
	s.recordHealth(game, id, out, time.Now())
	return out, nil
}

// RepairProfile applies the chosen findings' repairs and returns the profile as it now stands. Findings that are
// repaired in the app, or that a check no longer reports, are skipped.
func (s *Service) RepairProfile(game, id string, findingIDs []string) (Profile, error) {
	found, err := s.ProfileHealth(game, id)
	if err != nil {
		return Profile{}, err
	}
	var restore, revert, names, drop []string
	recoverJournals := false
	for _, f := range found {
		if !slices.Contains(findingIDs, f.ID) {
			continue
		}
		switch f.Repair {
		case RepairRestore:
			restore = append(restore, f.ID[len(HealthDrift)+1:])
			names = append(names, f.Items...)
		case RepairRevert:
			revert = append(revert, f.ID[len(HealthDrift)+1:])
			names = append(names, f.Items...)
		case RepairDropSnapshot:
			drop = append(drop, f.ID[len(HealthSnapshot)+1:])
		case RepairRecover:
			recoverJournals = true
		}
	}
	var errs []error
	if recoverJournals && s.HealthRecover != nil {
		errs = append(errs, s.HealthRecover(game))
	}
	restored := 0
	for _, key := range restore {
		if _, err := s.RestoreDriftEntry(game, id, key); err != nil {
			errs = append(errs, err)
			continue
		}
		restored++
	}
	// A changed mod is reverted, which keeps the config and data files its folder gained; only a deleted one is
	// restored whole.
	for _, key := range revert {
		if _, err := s.RevertDriftEntry(game, id, key); err != nil {
			errs = append(errs, err)
			continue
		}
		restored++
	}
	if restored > 0 {
		label := "Restored " + names[0] + " from the store"
		if restored > 1 {
			label = fmt.Sprintf("Restored %d mods from the store", restored)
		}
		errs = append(errs, s.store.recordSnapshot(game, id, historyRestored, label, restored))
	}
	for _, ev := range drop {
		errs = append(errs, s.store.dropHistoryEvent(game, id, ev))
	}
	// What was repaired changes what the next check finds, so the badge is refreshed by the scheduler.
	s.FlagHealth(game)
	p, err := s.store.read(game, id)
	return p, errors.Join(append(errs, err)...)
}

func (s *Store) missingFindings(game string, p Profile) ([]HealthFinding, error) {
	var out []HealthFinding
	for _, e := range p.Entries {
		missing, err := s.missingStoreKeys(game, []Entry{{Key: e.Key, ExtraStoreKeys: e.ExtraStoreKeys}})
		if err != nil {
			return nil, err
		}
		if len(missing) == 0 {
			continue
		}
		f := HealthFinding{
			ID: HealthMissing + ":" + e.Key, Kind: HealthMissing, Items: []string{entryLabel(e)},
			Entries: []Entry{e},
		}
		if fetchable(e.Source) {
			f.Repair = RepairDownload
		}
		out = append(out, f)
	}
	return out, nil
}

// fetchable matches the sources the app's download wants are built from.
func fetchable(src Source) bool {
	return (src.Kind == "nexus" && src.ModID > 0 && src.FileID > 0) || (src.Kind == "github" && src.Repo != "")
}

func driftFindings(p Profile, drift []Drift) []HealthFinding {
	var out []HealthFinding
	for _, d := range drift {
		if d.Key == "" || d.Kind == DriftUnknown {
			continue
		}
		i := entryIndex(p.Entries, d.Key)
		if i < 0 {
			continue
		}
		repair := RepairRestore
		if d.Kind == DriftChanged {
			repair = RepairRevert
		}
		out = append(out, HealthFinding{
			ID: HealthDrift + ":" + d.Key, Kind: HealthDrift, Cause: string(d.Kind),
			Items: []string{entryLabel(p.Entries[i])}, Repair: repair,
		})
	}
	return out
}

func unusedFinding(game string, unused []store.Item) (HealthFinding, bool) {
	if len(unused) == 0 {
		return HealthFinding{}, false
	}
	items := make([]string, len(unused))
	for i, it := range unused {
		items[i] = it.Key
		if it.Name != "" {
			items[i] = it.Name
			if it.Version != "" {
				items[i] += " " + it.Version
			}
		}
	}
	return HealthFinding{ID: HealthUnused + ":" + game, Kind: HealthUnused, Items: items, Repair: RepairCleanup}, true
}

// snapshotFindings lists history events whose saved mods cannot be read, or whose saved config files are gone.
func (s *Store) snapshotFindings(game, id string) ([]HealthFinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadHistory(game, id)
	if err != nil {
		return nil, err
	}
	var out []HealthFinding
	for _, ev := range data.Events {
		cause := ""
		if _, ok := snapshotEntries(&data, ev.SnapshotID); !ok {
			cause = "unreadable"
		} else if missingConfigCapture(data.dir, ev.SnapshotID) {
			cause = "configs"
		}
		if cause != "" {
			out = append(out, HealthFinding{
				ID: HealthSnapshot + ":" + ev.ID, Kind: HealthSnapshot, Cause: cause, Items: []string{ev.Label},
				At: ev.At, Repair: RepairDropSnapshot,
			})
		}
	}
	return out, nil
}

// missingConfigCapture reports a config index that cannot be read or names a file body that is gone. An event with
// no index captured no configs, which is not a fault.
func missingConfigCapture(dir, snapshotID string) bool {
	idx, err := readSnapshotIndex(dir, snapshotID)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		return true
	}
	for _, files := range idx {
		for _, hash := range files {
			if _, err := os.Stat(filepath.Join(dir, historyFilesDir, historyBlobsDir, hash)); err != nil {
				return true
			}
		}
	}
	return false
}

// dropHistoryEvent removes one event from the history; the profile itself is left as it is.
func (s *Store) dropHistoryEvent(game, id, eventID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadHistory(game, id)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(data.Events, func(ev HistoryEvent) bool { return ev.ID == eventID })
	if i < 0 {
		return nil
	}
	data.Events = slices.Delete(data.Events, i, i+1)
	if i < len(data.Events) {
		if next, ok := snapshotEntries(&data, data.Events[i].SnapshotID); ok {
			countEvent(&data, i, next)
		}
	}
	return writeHistory(data.dir, data, s.historyKeep())
}
