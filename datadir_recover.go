package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

type locationChoice int

const (
	choiceQuit locationChoice = iota
	choiceNoDialog
	choiceDefault
	choiceChoose
)

// dataDirOrRecover is datadir.Dir with the startup failures a user can fix. A relocated data folder that is not
// there (an unplugged drive) asks whether to pick another folder or use the default; either answer is saved as the
// new data-location. Any other failure is shown once and returned. Every failure is also written to startup.log in
// the default location, since the data folder, and with it mortar.log, may be the thing that is missing.
func dataDirOrRecover() (string, error) {
	for {
		dir, err := datadir.Dir()
		if err == nil {
			return dir, nil
		}
		logStartupFailure(err)
		missing, ok := errors.AsType[*datadir.MissingLocationError](err)
		if !ok {
			showStartupError(err.Error())
			return "", err
		}
		switch askMissingLocation(missing.Path) {
		case choiceDefault:
			if err := datadir.UseDefaultLocation(); err != nil {
				return "", err
			}
		case choiceChoose:
			picked := pickDataFolder()
			if picked == "" {
				continue
			}
			if err := datadir.SetLocation(picked); err != nil {
				return "", err
			}
		case choiceNoDialog:
			return "", err
		default:
			os.Exit(0)
		}
	}
}

func logStartupFailure(err error) {
	def, derr := datadir.DefaultDir()
	if derr != nil || os.MkdirAll(def, 0o700) != nil {
		return
	}
	f, ferr := fsx.OpenFile(filepath.Join(def, "startup.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if ferr != nil {
		return
	}
	_, _ = fmt.Fprintf(f, "%s %v\n", time.Now().Format(time.RFC3339), err)
	_ = f.Close()
}

// singleInstanceID is one id for the default data folder, so nxm and share handoffs reach the installed app, and one
// per data folder for a portable copy, so it can run beside the installed one.
func singleInstanceID(dataDir string) string {
	const id = "tech.rethunk.mortar"
	if !datadir.Portable() {
		return id
	}
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(dataDir))))
	return id + "-" + hex.EncodeToString(sum[:4])
}
