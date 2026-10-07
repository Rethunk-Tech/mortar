package profile

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RepairFolderRecords points every mod record whose folder is gone from mods/ at the sibling that name-repair
// (archive.RepairNames) renamed it to. A record written before zip names were decoded holds the bytes it could not read as
// U+FFFD, so the match is the same name with each U+FFFD run standing for one or more real characters, and it counts
// only when exactly one folder fits. Records whose folder exists, or that match nothing, stay as they are. It returns
// how many records changed; each profile.json is rewritten atomically, so a kill leaves it old or new.
func (s *Store) RepairFolderRecords() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dirs, err := filepath.Glob(filepath.Join(s.root, "*", "*", fileName))
	if err != nil {
		return 0, err
	}
	total := 0
	for _, path := range dirs {
		dir := filepath.Dir(path)
		id := filepath.Base(dir)
		if !idPattern.MatchString(id) {
			continue
		}
		p, err := readAt(dir, id)
		if err != nil {
			continue
		}
		n := repairFolderRecords(&p, filepath.Join(dir, "mods"))
		if n == 0 {
			continue
		}
		if err := writeProfile(dir, p); err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func repairFolderRecords(p *Profile, modsDir string) int {
	n := 0
	for i := range p.Entries {
		e := &p.Entries[i]
		if !e.hasFolder() {
			continue
		}
		for j := range e.Mods {
			c := &e.Mods[j]
			if !strings.ContainsRune(c.Folder, '�') {
				continue
			}
			plain, dotted, err := ModPaths(modsDir, e.Key, c.Folder)
			if err != nil || exists(plain) || exists(dotted) {
				continue
			}
			if fixed, ok := resolveFolder(filepath.Join(modsDir, e.Key), c.Folder); ok {
				c.Folder = fixed
				n++
			}
		}
	}
	return n
}

// resolveFolder walks the slash path folder under root, replacing each segment holding U+FFFD with the one existing
// folder it matches. The last segment may exist under its dotted (disabled) name; the record keeps the plain name.
func resolveFolder(root, folder string) (string, bool) {
	segs := strings.Split(folder, "/")
	dir := root
	for i, seg := range segs {
		last := i == len(segs)-1
		name, ok := matchSegment(dir, seg, last)
		if !ok {
			return "", false
		}
		segs[i] = name
		dir = filepath.Join(dir, name)
		if last {
			break
		}
	}
	return strings.Join(segs, "/"), true
}

func matchSegment(dir, seg string, dotted bool) (string, bool) {
	if !strings.ContainsRune(seg, '�') {
		if exists(filepath.Join(dir, seg)) || (dotted && exists(filepath.Join(dir, "."+seg))) {
			return seg, true
		}
		return "", false
	}
	parts := strings.FieldsFunc(seg, func(r rune) bool { return r == '�' })
	for i := range parts {
		parts[i] = regexp.QuoteMeta(parts[i])
	}
	lead, trail := "", ""
	if strings.HasPrefix(seg, "�") {
		lead = ".+"
	}
	if strings.HasSuffix(seg, "�") {
		trail = ".+"
	}
	re, err := regexp.Compile("^" + lead + strings.Join(parts, ".+") + trail + "$")
	if err != nil {
		return "", false
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	found := ""
	for _, ent := range ents {
		name := ent.Name()
		plain := name
		if dotted {
			plain = strings.TrimPrefix(name, ".")
		}
		if !re.MatchString(plain) {
			continue
		}
		if found != "" && found != plain {
			return "", false
		}
		found = plain
	}
	return found, found != ""
}
