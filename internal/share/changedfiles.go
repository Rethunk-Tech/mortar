package share

import (
	"errors"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// changedPrefix is where a .mortar file keeps a profile's changed copies (a file a mod rewrote or created while a
// game ran), by their path in the profile, beside the mod config files and under the same include switch and caps.
const changedPrefix = "changed/"

// ChangedFile is one changed copy: Path is slash separated from the profile's folder, inside its content folder or its
// changed/ folder.
type ChangedFile struct {
	Path string `json:"path"`
	Data []byte `json:"data"`
}

var unsafeName = regexp.MustCompile(`[<>:"|?*\\\x00-\x1f]`)

// validChangedPath accepts a clean relative slash path of ordinary file names, short enough to write on any system,
// inside the game's content folder or changed/. The receiving side checks it again before it writes.
func validChangedPath(game, p string) bool {
	if len(p) > maxRelPath || path.Clean(p) != p || !utf8.ValidString(p) || !profile.ValidChangedPath(game, p) {
		return false
	}
	for seg := range strings.SplitSeq(p, "/") {
		if len(seg) > 2*maxSegment || unsafeName.MatchString(seg) || reserved.MatchString(seg) || strings.HasSuffix(seg, ".") || strings.HasSuffix(seg, " ") {
			return false
		}
	}
	return true
}

// readChangedFiles collects the profile's changed copies; a file over the per-file cap or with a name that cannot be
// written elsewhere is listed as skipped, never dropped silently.
func readChangedFiles(profileDir, gameID string) (found []ChangedFile, skipped []string, err error) {
	rels, err := profile.ChangedFilesIn(gameID, profileDir)
	if err != nil {
		return nil, nil, err
	}
	for _, rel := range rels {
		if !validChangedPath(gameID, rel) {
			skipped = append(skipped, rel)
			continue
		}
		data, err := readCapped(filepath.Join(profileDir, filepath.FromSlash(rel)))
		if errors.Is(err, fsx.ErrTooLarge) {
			skipped = append(skipped, rel)
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		found = append(found, ChangedFile{Path: rel, Data: data})
	}
	return found, skipped, nil
}

func changedFiles(p Preview) map[string]string {
	out := make(map[string]string, len(p.ChangedFiles))
	for _, c := range p.ChangedFiles {
		out[c.Path] = string(c.Data)
	}
	return out
}
