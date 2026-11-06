package devsvc

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/go-safefs"
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

// trashFailed names the programs that hold dir when it could not be moved, so the player knows what to close. The
// error is untagged on purpose: a permission kind would show a sentence about Mortar's own permissions instead.
func trashFailed(dir string, err error) error {
	if !safefs.IsDir(dir) {
		return err
	}
	names, ok := holderNames(dir)
	if !ok {
		return err
	}
	log.Printf("devsvc: moving %s to the trash: %v; held by %v", dir, err, names)
	return inUse(dir, names)
}

// inUse is the error for a folder Windows would not move, naming what holds it when that is known.
func inUse(dir string, names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("could not move %s to the Recycle Bin because Windows denied access. Close programs using the folder, such as an editor or a terminal open in it, then try again", dir)
	}
	them := "them"
	if len(names) == 1 {
		them = "it"
	}
	list := names[0]
	if n := len(names); n > 1 {
		list = strings.Join(names[:n-1], ", ") + " and " + names[n-1]
	}
	return fmt.Errorf("%s is in use by %s. Close %s, then try again", filepath.Base(dir), list, them)
}
