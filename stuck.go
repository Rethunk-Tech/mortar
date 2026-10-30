package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Rethunk-Tech/go-gameproc"

	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/i18n"
)

// stuckWait is how long a running Mortar may take to answer before this start offers to end it.
const stuckWait = 10 * time.Second

// endStuckInstance offers to end a Mortar that holds the data folder but has stopped answering, so this start goes on
// instead of handing its arguments to a frozen window and closing without a word.
func endStuckInstance(dataDir string) {
	pid, stuck := controlwire.Unanswering(dataDir, stuckWait)
	if !stuck || pid == os.Getpid() || !runsMortar(pid) {
		return
	}
	log.Printf("mortar (pid %d) holds the data folder but has not answered for %s", pid, stuckWait)
	if !askEndStuck(i18n.T("Mortar is already running but has not answered for {count} seconds.\n\n"+
		"End it and start Mortar again? Anything it was doing stops.", "count", strconv.Itoa(int(stuckWait/time.Second)))) {
		return
	}
	p, err := os.FindProcess(pid)
	if err == nil {
		err = p.Kill()
	}
	if err != nil {
		log.Printf("end mortar (pid %d): %v", pid, err)
		return
	}
	waitGone(p)
	log.Printf("ended mortar (pid %d)", pid)
}

// runsMortar reports whether pid runs a program with this one's name, so a pid the system gave to another program
// after Mortar exited is never ended.
func runsMortar(pid int) bool {
	self, err := os.Executable()
	if err != nil {
		return false
	}
	exe, err := gameproc.ImagePath(pid)
	return err == nil && filepath.Base(exe) == filepath.Base(self)
}
