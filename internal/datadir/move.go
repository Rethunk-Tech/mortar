package datadir

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

var (
	ErrInside   = errors.New("the new folder is inside the current data folder")
	ErrNotEmpty = errors.New("the new folder is not empty")
	ErrNoSpace  = errors.New("not enough free space")
)

// SpaceError says how many bytes the move needs.
type SpaceError struct {
	Need int64
}

type RelocateEstimate struct {
	Bytes     int64 `json:"bytes"`
	FreeBytes int64 `json:"freeBytes"`
}

// ExtentSizer measures a tree's physical bytes with shared extents counted once.
type ExtentSizer func(root string) (int64, error)

// EstimateRelocate sizes a move of src to dest. Where dest can clone files the move keeps profile files that clone
// the store shared, so the need is shared's extent-aware total. Elsewhere hard-linked files count once only when dest
// can hold links; FAT, exFAT and SMB targets refuse them, so each name is copied in full there.
func EstimateRelocate(src, dest string, shared ExtentSizer) (RelocateEstimate, error) {
	where := dest
	free, err := FreeBytes(where)
	if err != nil {
		where = filepath.Dir(dest)
		if free, err = FreeBytes(where); err != nil {
			return RelocateEstimate{}, err
		}
	}
	var need int64
	if shared != nil && clonesSupported(where) {
		need, err = shared(src)
	} else {
		need, err = size(src, linksSupported(where))
	}
	if err != nil {
		return RelocateEstimate{}, err
	}
	return RelocateEstimate{Bytes: need, FreeBytes: free}, nil
}

// clonesSupported probes dir with a real clone. A dir that cannot be probed is assumed not to support them.
func clonesSupported(dir string) bool {
	a, err := os.CreateTemp(dir, ".mortar-clone-*")
	if err != nil {
		return false
	}
	_, _ = a.WriteString("probe")
	_ = a.Close()
	defer func() { _ = os.Remove(a.Name()) }()
	b := a.Name() + ".c"
	if cloneFile(a.Name(), b) != nil {
		return false
	}
	_ = os.Remove(b)
	return true
}

// linksSupported probes dir with a real hard link. A dir that cannot be probed is assumed to support them.
func linksSupported(dir string) bool {
	a, err := os.CreateTemp(dir, ".mortar-link-*")
	if err != nil {
		return true
	}
	_ = a.Close()
	defer func() { _ = os.Remove(a.Name()) }()
	b := a.Name() + ".l"
	if os.Link(a.Name(), b) != nil {
		return false
	}
	_ = os.Remove(b)
	return true
}

func (e *SpaceError) Error() string {
	return fmt.Sprintf("not enough free space: this move needs about %d MB free", (e.Need+1024*1024-1)/(1024*1024))
}

func (e *SpaceError) Unwrap() error { return ErrNoSpace }

// copied is one source file already written to the target: where, and its SHA-256.
type copied struct{ to, sum string }

// relocator copies a data folder file by file and keeps the sharing the source had:
//   - further names of a hard-linked inode become links;
//   - on the same clone-capable filesystem every file is cloned, so files that shared extents still do;
//   - across filesystems, a profile file that holds the same bytes as its store file becomes a clone of the store
//     file already copied (the store goes first), so profiles stay shared on a ReFS or btrfs target;
//   - anything else is copied while its SHA-256 is taken, so verifyCopy only has to read the target.
type relocator struct {
	dest string
	// sums maps each source file's slash path to its SHA-256, or "" for a same-filesystem clone, which shares the
	// source's extents and is only checked by length.
	sums    map[string]string
	sizes   map[string]int64
	linked  map[fileKey]copied
	noClone bool
	// same reports whether a file and its target folder share a filesystem; tests replace it.
	same func(from, to string) bool
}

func newRelocator(dest string) *relocator {
	return &relocator{dest: dest, sums: map[string]string{}, sizes: map[string]int64{}, linked: map[fileKey]copied{}, same: sameFS}
}

// put is the copyTree callback; rel is relative to the data folder.
func (r *relocator) put(from, to, rel string) error {
	info, err := os.Stat(from)
	if err != nil {
		return err
	}
	slash := filepath.ToSlash(rel)
	r.sizes[slash] = info.Size()
	key, linked := linkedKey(from, info)
	if linked {
		if first, ok := r.linked[key]; ok && os.Link(first.to, to) == nil {
			r.sums[slash] = first.sum
			return nil
		}
	}
	sum, err := r.place(from, to, slash, info.Size())
	if err != nil {
		return err
	}
	r.sums[slash] = sum
	if _, ok := r.linked[key]; linked && !ok {
		r.linked[key] = copied{to, sum}
	}
	return nil
}

// place writes one file, cloning where it can; it returns the source's SHA-256, or "" for a clone of the source.
func (r *relocator) place(from, to, rel string, size int64) (string, error) {
	if !r.noClone {
		if r.same(from, to) {
			switch err := cloneFile(from, to); {
			case err == nil:
				return "", nil
			case !isLinkFallback(err):
				return "", err
			}
			r.noClone = true
		} else if sum, err := r.cloneFromStore(from, to, rel, size); err != nil || sum != "" {
			return sum, err
		}
	}
	return copyHashing(from, to)
}

func sameFS(from, to string) bool {
	a, errA := fileDevice(from)
	b, errB := fileDevice(filepath.Dir(to))
	return errA == nil && errB == nil && a == b
}

// cloneFromStore clones the target's copy of the store file that rel's content came from. It returns the source's
// sum when it did, and "" when the file has no store twin with identical bytes or the target cannot clone.
func (r *relocator) cloneFromStore(from, to, rel string, size int64) (string, error) {
	for _, twin := range storeTwins(rel) {
		want, ok := r.sums[twin]
		if !ok || want == "" || r.sizes[twin] != size {
			continue
		}
		sum, err := fsx.SHA256(from)
		if err != nil {
			return "", err
		}
		if sum != want {
			return "", nil
		}
		err = cloneFile(filepath.Join(r.dest, filepath.FromSlash(twin)), to)
		if err == nil {
			return sum, nil
		}
		if !isLinkFallback(err) {
			return "", err
		}
		r.noClone = true
		return "", nil
	}
	return "", nil
}

// storeTwins lists the store files a profile file may have been materialised from:
// profiles/<game>/<id>/mods/<key>/<file> comes from store/<game>/<key>/<file> (the folder is dotted while the mod
// is off), and a multi-file mod's extra items sit one level deeper, as mods/<key>/<extra key>/<file>.
func storeTwins(rel string) []string {
	parts := strings.Split(rel, "/")
	if len(parts) < 6 || parts[0] != "profiles" || parts[3] != "mods" {
		return nil
	}
	game, key, file := parts[1], strings.TrimPrefix(parts[4], "."), parts[5:]
	twins := []string{"store/" + game + "/" + key + "/" + strings.Join(file, "/")}
	if len(file) > 1 {
		twins = append(twins, "store/"+game+"/"+file[0]+"/"+strings.Join(file[1:], "/"))
	}
	return twins
}

// copyHashing is CopyFile that also returns the hex SHA-256 of what it read.
func copyHashing(src, dst string) (sum string, err error) {
	in, err := fsx.Open(src)
	if err != nil {
		return "", err
	}
	defer func() { _ = in.Close() }()
	out, err := fsx.CreateExcl(dst, 0o600)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, out.Close()) }()
	h := sha256.New()
	if _, err = io.Copy(out, io.TeeReader(in, h)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// verifyCopy hashes each target file and compares it with the sum taken while the source was read; a clone of the
// source has no sum and is compared by length.
func (r *relocator) verifyCopy() error {
	for rel, want := range r.sums {
		path := filepath.Join(r.dest, filepath.FromSlash(rel))
		if want == "" {
			st, err := os.Stat(path)
			if err != nil {
				return err
			}
			if st.Size() != r.sizes[rel] {
				return fmt.Errorf("copy of %s did not match", rel)
			}
			continue
		}
		got, err := fsx.SHA256(path)
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("copy of %s did not match", rel)
		}
	}
	return nil
}

// clampProgress keeps reported bytes within the total, which counts each hard-linked inode once.
func clampProgress(report func(CopyProgress)) func(CopyProgress) {
	if report == nil {
		return nil
	}
	return func(p CopyProgress) {
		p.Bytes = min(p.Bytes, p.TotalBytes)
		report(p)
	}
}

func emptyDir(path string) error {
	ents, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(path, 0o700)
		}
		return err
	}
	if len(ents) > 0 {
		return ErrNotEmpty
	}
	return nil
}

func removeOld(src, def string) error {
	if fsx.SamePath(src, def) {
		ents, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range ents {
			if e.Name() == PointerName {
				continue
			}
			if err := fsx.RemoveAll(filepath.Join(src, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	return fsx.RemoveAll(src)
}

// Relocate copies src into dest (which must be empty), verifies the copy, writes the pointer at def, then removes src.
// shared sizes the copy where dest can clone files (nil sizes it by file length); the store is copied first so the
// profile files that clone it can clone it again.
func Relocate(src, dest, def string, shared ExtentSizer, reports ...func(CopyProgress)) error {
	src, dest, def = filepath.Clean(src), filepath.Clean(dest), filepath.Clean(def)
	var report func(CopyProgress)
	if len(reports) > 0 {
		report = reports[0]
	}
	if UnderRoot(src, dest) {
		return ErrInside
	}
	if err := emptyDir(dest); err != nil {
		return err
	}
	estimate, err := EstimateRelocate(src, dest, shared)
	if err != nil {
		return err
	}
	if estimate.FreeBytes < estimate.Bytes {
		return &SpaceError{Need: estimate.Bytes}
	}
	r := newRelocator(dest)
	skipped, err := copyTreeFirst(src, dest, "store", clampProgress(report), r.put)
	if err != nil {
		_ = fsx.RemoveAll(dest)
		return err
	}
	if len(skipped) > 0 {
		slog.Warn("data move left linked folders behind", "folders", skipped)
	}
	if err := r.verifyCopy(); err != nil {
		_ = fsx.RemoveAll(dest)
		return err
	}
	if err := os.MkdirAll(def, 0o700); err != nil {
		return err
	}
	if err := WriteFile(filepath.Join(def, PointerName), []byte(dest+"\n"), 0o600); err != nil {
		return err
	}
	return removeOld(src, def)
}
