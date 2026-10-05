package packsvc

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
)

const (
	backupDoc     = "backup.json"
	backupVersion = 1
	backupPrefix  = "files/"
	// maxBackupBytes caps a backup file and everything it unpacks to, counted as read.
	maxBackupBytes = 256 << 20
)

type backupJSON struct {
	Version int             `json:"version"`
	Game    string          `json:"game"`
	Profile profile.Profile `json:"profile"`
}

// RestoreResult is the profile a backup made and what it queued. Unavailable names the mods that cannot be downloaded
// again (one installed from a file on disk), which the profile keeps but cannot use until they are installed again.
type RestoreResult struct {
	Game        string   `json:"game"`
	Profile     string   `json:"profile"`
	Name        string   `json:"name"`
	Queued      int      `json:"queued"`
	Unavailable []string `json:"unavailable"`
}

// Backup writes the profile to dest as one file: backup.json (the format version, the game and every setting and
// entry of the profile) and under files/ its history, config files and cover. Mod files are left out; a restore
// downloads them again.
func (s *Service) Backup(gameID, profileID, dest string) error {
	p, files, err := s.Profiles.Backup(gameID, profileID)
	if err != nil {
		return err
	}
	doc, err := json.MarshalIndent(backupJSON{Version: backupVersion, Game: gameID, Profile: p}, "", "  ")
	if err != nil {
		return err
	}
	out := map[string][]byte{backupDoc: doc}
	for rel, data := range files {
		out[backupPrefix+rel] = data
	}
	return writeZip(dest, out)
}

// BackupDialog asks where to save, then backs up as Backup does; an empty path means the player cancelled.
func (s *Service) BackupDialog(gameID, profileID string) (string, error) {
	if s.App == nil {
		return "", errors.New("no window to ask where to save")
	}
	p, err := s.find(gameID, profileID)
	if err != nil {
		return "", err
	}
	d := s.App.Dialog.SaveFile().SetFilename(packageName(p.Name)+".mortar-backup.zip").AddFilter("Mortar profile backup (zip)", "*.zip")
	d.AttachToWindow(s.App.Window.Current())
	dest, err := d.PromptForSingleSelection()
	if err != nil || dest == "" {
		return "", err
	}
	return dest, s.Backup(gameID, profileID, dest)
}

// Restore makes a new profile from the backup at path and queues the downloads of the mods this computer does not
// have. gameID may be empty; when given it must be the backup's game.
func (s *Service) Restore(ctx context.Context, path, gameID string) (RestoreResult, error) {
	doc, files, err := readBackup(path)
	if err != nil {
		return RestoreResult{}, err
	}
	if gameID != "" && gameID != doc.Game {
		return RestoreResult{}, fmt.Errorf("this backup is of a %s profile, not %s", doc.Game, gameID)
	}
	p, missing, err := s.Profiles.RestoreBackup(doc.Game, doc.Profile, files)
	if err != nil {
		return RestoreResult{}, err
	}
	res := RestoreResult{Game: doc.Game, Profile: p.ID, Name: p.Name, Unavailable: []string{}}
	var reqs []queue.Request
	for _, e := range missing {
		if r, ok := restoreRequest(doc.Game, p.ID, e); ok {
			reqs = append(reqs, r)
		} else {
			res.Unavailable = append(res.Unavailable, entryName(e))
		}
	}
	if len(reqs) == 0 {
		return res, nil
	}
	items, err := s.Queue.Add(ctx, reqs)
	res.Queued = len(items)
	return res, err
}

// RestoreDialog asks for a backup file, then restores it as Restore does; an empty Profile means the player cancelled.
func (s *Service) RestoreDialog(ctx context.Context, gameID string) (RestoreResult, error) {
	if s.App == nil {
		return RestoreResult{}, errors.New("no window to ask for the file")
	}
	d := s.App.Dialog.OpenFile().AddFilter("Mortar profile backup (zip)", "*.zip")
	d.AttachToWindow(s.App.Window.Current())
	path, err := d.PromptForSingleSelection()
	if err != nil || path == "" {
		return RestoreResult{Unavailable: []string{}}, err
	}
	return s.Restore(ctx, path, gameID)
}

// restoreRequest is the download that puts the entry's store item back: the same file from the same source, which
// lands under the same store key, so the entry the restore already wrote picks it up.
func restoreRequest(game, profileID string, e profile.Entry) (queue.Request, bool) {
	src := e.Source
	r := queue.Request{Kind: queue.KindInstall, Game: game, Profile: profileID, Name: src.Name, Version: src.Version, Disabled: e.Disabled, Fomod: e.Fomod}
	switch src.Kind {
	case profile.KindThunderstore:
		r.Package = src.Name
	case profile.KindModrinth, profile.KindItch:
		r.Package, r.Source = src.Name, src.Kind
	case profile.KindGitHub:
		r.Repo, r.Tag, r.Asset, r.FileName = src.Repo, src.Tag, src.Asset, src.Asset
	case profile.KindNexus:
		r.ModID, r.FileID, r.FileName = src.ModID, src.FileID, src.Name
		if e.IsOverlay() {
			r.Overlay = &queue.OverlayPlace{From: e.OverlayFrom, To: e.OverlayTo, Off: e.OverlayOff}
		}
	default:
		return queue.Request{}, false
	}
	return r, true
}

func readBackup(path string) (backupJSON, map[string][]byte, error) {
	if st, err := os.Stat(path); err != nil {
		return backupJSON{}, nil, err
	} else if st.Size() > maxBackupBytes {
		return backupJSON{}, nil, errors.New("the backup is too large")
	}
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return backupJSON{}, nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return backupJSON{}, nil, fmt.Errorf("not a Mortar profile backup: %w", err)
	}
	files := map[string][]byte{}
	var doc []byte
	var total int64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return backupJSON{}, nil, err
		}
		data, err := io.ReadAll(io.LimitReader(rc, maxBackupBytes-total+1))
		_ = rc.Close()
		if err != nil {
			return backupJSON{}, nil, err
		}
		if total += int64(len(data)); total > maxBackupBytes {
			return backupJSON{}, nil, errors.New("the backup unpacks to too much")
		}
		switch rel, ok := strings.CutPrefix(f.Name, backupPrefix); {
		case f.Name == backupDoc:
			doc = data
		case ok:
			files[rel] = data
		default:
			return backupJSON{}, nil, fmt.Errorf("the backup holds %q, which Mortar does not write", f.Name)
		}
	}
	var b backupJSON
	if doc == nil {
		return backupJSON{}, nil, errors.New("not a Mortar profile backup: no " + backupDoc)
	}
	if err := json.Unmarshal(doc, &b); err != nil {
		return backupJSON{}, nil, fmt.Errorf("%s: %w", backupDoc, err)
	}
	if b.Version != backupVersion {
		return backupJSON{}, nil, fmt.Errorf("this backup is format %d; this Mortar reads format %d", b.Version, backupVersion)
	}
	if b.Game == "" {
		return backupJSON{}, nil, errors.New(backupDoc + " names no game")
	}
	return b, files, nil
}
