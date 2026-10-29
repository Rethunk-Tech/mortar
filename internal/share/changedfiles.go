package share

import (
	"errors"
	"log/slog"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/winname"
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
	return profile.ChangedTypeAllowed(p) && validChangedLocation(game, p) && winSafe(p)
}

// validChangedLocation is validChangedPath without the file type check, which a file from an older release may fail
// on a path that is otherwise safe.
func validChangedLocation(game, p string) bool {
	if len(p) > maxRelPath || path.Clean(p) != p || !utf8.ValidString(p) || !profile.ValidChangedLocation(game, p) {
		return false
	}
	for seg := range strings.SplitSeq(p, "/") {
		if len(seg) > 2*maxSegment || unsafeName.MatchString(seg) || strings.HasSuffix(seg, ".") || strings.HasSuffix(seg, " ") {
			return false
		}
	}
	return true
}

// deviceSegment is a segment whose only fault is a Windows device name such as CON.
func deviceSegment(seg string) bool {
	return !winname.Valid(seg) && !unsafeName.MatchString(seg) && !strings.HasSuffix(seg, ".") && !strings.HasSuffix(seg, " ")
}

// hasDeviceSegment reports that p has a segment that is a device name.
func hasDeviceSegment(p string) bool {
	return slices.ContainsFunc(strings.Split(p, "/"), deviceSegment)
}

// devicesRenamed is p with each device-name segment made plain, so the rest of its checks can run on any OS.
func devicesRenamed(p string) string {
	segs := strings.Split(p, "/")
	for i, seg := range segs {
		if deviceSegment(seg) {
			segs[i] = "_"
		}
	}
	return strings.Join(segs, "/")
}

// readChangedFiles collects the profile's changed copies; a file over the per-file cap or with a name that cannot be
// written elsewhere is listed as skipped, never dropped silently.
func readChangedFiles(profileDir, gameID string) (found []ChangedFile, skipped []string, err error) {
	rels, err := profile.ChangedFilesIn(gameID, profileDir)
	if err != nil {
		return nil, nil, err
	}
	// Read refuses two entries differing only by case, so a case variant of an earlier name is left out.
	folded := map[string]bool{}
	for _, rel := range rels {
		if !validChangedPath(gameID, rel) {
			skipped = append(skipped, rel)
			continue
		}
		if folded[strings.ToLower(rel)] {
			slog.Warn("share: skipped a changed copy whose name differs from an earlier one only by case", "path", rel)
			skipped = append(skipped, rel)
			continue
		}
		folded[strings.ToLower(rel)] = true
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
