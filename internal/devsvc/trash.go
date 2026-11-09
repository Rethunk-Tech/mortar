package devsvc

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/go-safefs"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// holdBuild waits for a build of the project in dir to finish and keeps the next from starting until the returned
// func is called.
func holdBuild(dir string) (release func()) {
	stored, _ := building.LoadOrStore(dir, &sync.Mutex{})
	mu, ok := stored.(*sync.Mutex)
	if !ok {
		return func() {}
	}
	mu.Lock()
	return mu.Unlock
}

// trashFailed names the programs that hold dir when Windows would not move it, so the player knows what to close. The
// error is tagged InUse, not Permission, whose sentence would blame Mortar's own permissions.
func trashFailed(dir string, err error) error {
	if !errors.Is(err, safefs.ErrInUse) || !safefs.IsDir(dir) {
		return err
	}
	holders, herr := safefs.Holders(projectPaths(dir))
	if herr != nil {
		slog.Info("devsvc: finding what holds a folder", "dir", dir, "err", herr)
	}
	names := []string{}
	for _, h := range holders {
		if h.Name != "" && !slices.Contains(names, h.Name) {
			names = append(names, h.Name)
		}
	}
	slog.Info("devsvc: moving to the trash", "dir", dir, "err", err, "names", names)
	return inUse(dir, names)
}

// inUseError is a folder Windows would not move. Its English text is for logs; Detail is what the GUI words its own
// sentence from, so the list of programs is joined in the player's language (usererr.Detailer).
type inUseError struct {
	Folder  string   `json:"folder"`
	Holders []string `json:"holders"`
}

func (e *inUseError) Error() string {
	if len(e.Holders) == 0 {
		return fmt.Sprintf("could not move %s to the Recycle Bin because Windows denied access. Close programs using the folder, such as an editor or a terminal open in it, then try again", e.Folder)
	}
	return fmt.Sprintf("%s is in use by: %s", filepath.Base(e.Folder), strings.Join(e.Holders, ", "))
}

func (e *inUseError) Detail() any { return e }

// inUse is the error for a folder Windows would not move, naming what holds it when that is known.
func inUse(dir string, names []string) error {
	return usererr.Wrap(usererr.InUse, &inUseError{Folder: dir, Holders: usererr.NonNil(names)})
}
