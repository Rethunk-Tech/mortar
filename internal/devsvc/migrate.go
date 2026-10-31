package devsvc

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"path/filepath"

	"github.com/Rethunk-Tech/go-safefs"
)

var errDamaged = errors.New("damaged record")

// MigrateRecords rewrites each Develop project's .mortar records to the shape a new project gets, so the repository
// carries the fields Mortar now reads from them, and drops the settings copies that moved there. A second run
// rewrites nothing. It touches only the two record files and reads no project code, so an untrusted project is
// migrated too.
func (s *Service) MigrateRecords() {
	var seen, wrote, skipped int
	cur := s.Settings.Get()
	for gameID := range cur.Games {
		for _, p := range cur.GamePrefs(gameID).DevProjects {
			seen++
			dir := filepath.Clean(p.Dir)
			for _, m := range []struct {
				name   string
				update func() ([]byte, error)
			}{
				{templateRecordFile, func() ([]byte, error) { return standardTemplate(dir, gameID, p.TemplateIcon) }},
				{assetRecordFile, func() ([]byte, error) { return standardRecord(dir, gameID) }},
			} {
				changed, err := migrateFile(dir, m.name, m.update)
				if err != nil {
					skipped++
					log.Printf("devsvc: leaving %s as it is: %v", filepath.Join(dir, ".mortar", m.name), err)
				} else if changed {
					wrote++
				}
			}
			if rec, ok := readTemplateRecord(dir); ok && rec.TemplateIcon != "" && p.TemplateIcon != "" {
				if err := s.Settings.ClearDevProjectTemplateIcon(gameID, p.Dir); err != nil {
					log.Printf("devsvc: clearing the icon kept for %s: %v", dir, err)
				}
			}
		}
	}
	log.Printf("devsvc: project records: %d projects, %d files rewritten, %d skipped", seen, wrote, skipped)
}

// migrateFile replaces .mortar/name with what update gives when the bytes differ; a missing file is not an error.
func migrateFile(dir, name string, update func() ([]byte, error)) (bool, error) {
	body, err := safefs.ReadFile(filepath.Join(dir, ".mortar", name))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	next, err := update()
	if err != nil || bytes.Equal(next, body) {
		return false, err
	}
	return true, writeTracked(dir, name, next)
}

// standardTemplate is template.json with the game and the icon hash, and every other key as it was.
func standardTemplate(dir, gameID, icon string) ([]byte, error) {
	body, err := safefs.ReadFile(filepath.Join(dir, ".mortar", templateRecordFile))
	if err != nil {
		return nil, err
	}
	var rec projectRecord
	var all map[string]json.RawMessage
	if json.Unmarshal(body, &rec) != nil || json.Unmarshal(body, &all) != nil {
		return nil, errDamaged
	}
	rec.Game = cmp.Or(rec.Game, gameID)
	rec.TemplateIcon = cmp.Or(rec.TemplateIcon, icon)
	std, err := recordFile(rec)
	if err != nil {
		return nil, err
	}
	var known map[string]json.RawMessage
	if err := json.Unmarshal(std, &known); err != nil {
		return nil, err
	}
	extra := false
	for k, v := range all {
		if _, ok := known[k]; !ok {
			known[k], extra = v, true
		}
	}
	if !extra {
		return std, nil
	}
	return recordFile(known)
}

// standardRecord is project.json as a new record is written: the game, and the kind and name only where the files do
// not give them.
func standardRecord(dir, gameID string) ([]byte, error) {
	rec, ok := readRawRecord(dir)
	if !ok {
		return nil, errDamaged
	}
	kind, name := migrateKind(gameID, rec.Kind), rec.Name
	rec.Game = cmp.Or(rec.Game, gameID)
	rec.Kind, rec.Name = "", ""
	if got, _ := rec.resolve(dir); got.Kind != kind {
		rec.Kind = kind
	}
	if got, _ := rec.resolve(dir); got.Name != name {
		rec.Name = name
	}
	if rec.Kind == kindCode && len(listCsproj(dir)) <= 1 {
		rec.Csproj = ""
	}
	return recordFile(rec)
}
