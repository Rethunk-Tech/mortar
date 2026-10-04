package savessvc

import (
	"fmt"
	"os"
	"slices"
)

// SaveUsage is the backup footprint of one save folder. One zip can hold several saves, so it counts toward each
// of them and the per-save bytes can add up to more than the total.
type SaveUsage struct {
	Save  string `json:"save"`
	Bytes int64  `json:"bytes"`
	Count int    `json:"count"`
}

// BackupsUsage is how much disk the save backups take.
type BackupsUsage struct {
	TotalBytes int64       `json:"totalBytes"`
	PerSave    []SaveUsage `json:"perSave"`
}

// TrimResult is what TrimBackups deleted.
type TrimResult struct {
	Removed    int   `json:"removed"`
	FreedBytes int64 `json:"freedBytes"`
}

// BackupsUsage sums backup sizes in total and per save. Backups exist for Stardew Valley only, so game is not read.
func (s *Service) BackupsUsage(_ string) (BackupsUsage, error) {
	list, err := s.ListBackups()
	if err != nil {
		return BackupsUsage{}, err
	}
	out := BackupsUsage{PerSave: []SaveUsage{}}
	at := map[string]int{}
	for _, b := range list {
		out.TotalBytes += b.Size
		for _, sv := range b.Saves {
			i, ok := at[sv.Folder]
			if !ok {
				i = len(out.PerSave)
				at[sv.Folder] = i
				out.PerSave = append(out.PerSave, SaveUsage{Save: sv.Folder})
			}
			out.PerSave[i].Bytes += b.Size
			out.PerSave[i].Count++
		}
	}
	slices.SortFunc(out.PerSave, func(a, b SaveUsage) int { return int(b.Bytes - a.Bytes) })
	return out, nil
}

// TrimBackups deletes backups that are not among the newest keepPerSave of any save they hold. Pinned backups are
// never deleted and do not count toward the limit; a backup whose saves cannot be read is kept.
func (s *Service) TrimBackups(_ string, keepPerSave int) (TrimResult, error) {
	if keepPerSave < 1 {
		return TrimResult{}, fmt.Errorf("keep at least 1 backup per save, got %d", keepPerSave)
	}
	if s.gameBusy() {
		return TrimResult{}, ErrBusy
	}
	reads, err := s.backupReads()
	if err != nil {
		return TrimResult{}, err
	}
	list, err := s.ListBackups()
	if err != nil {
		return TrimResult{}, err
	}
	kept := map[string]bool{}
	seen := map[string]int{}
	for _, b := range list { // newest first
		if b.Pinned || len(b.Saves) == 0 {
			kept[b.Name] = true
			continue
		}
		for _, sv := range b.Saves {
			if seen[sv.Folder] < keepPerSave {
				kept[b.Name] = true
			}
			seen[sv.Folder]++
		}
	}
	var out TrimResult
	for _, b := range list {
		if kept[b.Name] {
			continue
		}
		if err := backupNameOK(b.Name); err != nil {
			return out, err
		}
		if err := os.Remove(findBackup(reads, b.Name)); err != nil {
			return out, err
		}
		out.Removed++
		out.FreedBytes += b.Size
	}
	return out, nil
}
