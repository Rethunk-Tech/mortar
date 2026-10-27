//go:build !windows

package fsx

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// Trash moves path into the user's freedesktop trash (XDG_DATA_HOME/Trash), where the file manager can restore it.
// A path on another filesystem (another drive, or another btrfs subvolume such as /var/tmp beside /home) is copied
// there and then deleted.
func Trash(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		data = filepath.Join(home, ".local", "share")
	}
	files, info := filepath.Join(data, "Trash", "files"), filepath.Join(data, "Trash", "info")
	for _, dir := range []string{files, info} {
		if err := os.MkdirAll(dir, 0o700); err != nil { // #nosec G703 -- the user's own trash, from XDG_DATA_HOME or home
			return err
		}
	}
	body := "[Trash Info]\nPath=" + (&url.URL{Path: abs}).EscapedPath() + "\nDeletionDate=" + time.Now().Format("2006-01-02T15:04:05") + "\n"
	base := filepath.Base(abs)
	for i := 1; ; i++ {
		name := base
		if i > 1 {
			name = base + "." + strconv.Itoa(i)
		}
		record := filepath.Join(info, name+".trashinfo")
		f, err := os.OpenFile(record, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G304 G703 -- a new record in the user's own trash
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return err
		}
		_, werr := f.WriteString(body)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr == nil {
			werr = moveAcross(abs, filepath.Join(files, name))
		}
		if werr != nil {
			_ = os.Remove(record) // #nosec G703 -- the record just written in the user's own trash
			return fmt.Errorf("moving %s to the trash: %w", abs, werr)
		}
		return nil
	}
}

// moveAcross renames from to to, or copies it and deletes the original when they are on different filesystems.
func moveAcross(from, to string) error {
	err := os.Rename(from, to) // #nosec G703 -- a folder the caller chose, into the user's own trash
	if !errors.Is(err, syscall.EXDEV) {
		return err
	}
	if err := copyAll(from, to); err != nil {
		_ = os.RemoveAll(to) // #nosec G703 -- the partial copy just made in the user's own trash
		return err
	}
	return os.RemoveAll(from) // #nosec G703 -- the folder the caller asked to trash, now copied into it
}

// copyAll copies a file or folder tree, keeping symlinks as links so the trashed copy restores as it was. Both
// sides go through os.Root, so a link inside the tree is never followed out of it.
func copyAll(from, to string) error {
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return copyRegular(from, to, info.Mode().Perm())
	}
	if err := os.Mkdir(to, info.Mode().Perm()|0o700); err != nil { // #nosec G703 -- inside the user's own trash
		return err
	}
	src, err := os.OpenRoot(from)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	dst, err := os.OpenRoot(to)
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()
	return fs.WalkDir(src.FS(), ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil || rel == "." {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return dst.Mkdir(rel, info.Mode().Perm()|0o700)
		case d.Type()&fs.ModeSymlink != 0:
			target, err := src.Readlink(rel)
			if err != nil {
				return err
			}
			return dst.Symlink(target, rel)
		case !info.Mode().IsRegular():
			return nil // a socket or pipe has nothing to keep, and reading one could block
		}
		in, err := src.Open(rel)
		if err != nil {
			return err
		}
		defer func() { _ = in.Close() }()
		out, err := dst.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
		if err != nil {
			return err
		}
		return copyInto(out, in)
	})
}

// copyRegular copies one file, for a trashed path that is a file rather than a folder.
func copyRegular(from, to string, perm fs.FileMode) error {
	in, err := os.Open(from) // #nosec G304 G703 -- the file the caller asked to trash
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm) // #nosec G304 G703 -- inside the user's own trash
	if err != nil {
		return err
	}
	return copyInto(out, in)
}

func copyInto(out *os.File, in io.Reader) error {
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
