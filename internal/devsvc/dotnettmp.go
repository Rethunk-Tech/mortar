package devsvc

import (
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
)

// The .NET CLI leaves two empty folders from Directory.CreateTempSubdirectory in the temp folder on every command, and
// ThunderPipe leaves its package staging folder after every Release build, so a session of builds on save fills /tmp.
// dotnet runs with its temp folder inside Mortar's cache instead, and each run clears what earlier runs left there.
// The folder itself stays, because MSBuild nodes and the compiler server outlive a build and keep using it.
var dotnetLeftover = regexp.MustCompile(`^[A-Za-z0-9]{6}$`)

// msbuildTemp is each MSBuild process's own folder; a node reused across builds keeps its folder while it lives.
var msbuildTemp = regexp.MustCompile(`^MSBuildTemp[A-Za-z0-9]{6}$`)

// dotnetLeftoverAge spares a folder a build running beside this one may still be filling; msbuildTempAge outlasts
// any idle MSBuild node.
const (
	dotnetLeftoverAge = 10 * time.Minute
	msbuildTempAge    = 24 * time.Hour
)

func (s *Service) dotnetEnv() []string {
	env := os.Environ()
	if s.CacheDir == "" {
		return env
	}
	dir := filepath.Join(s.CacheDir, "dotnet-tmp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return env
	}
	now := time.Now()
	sweepDotnetTemp(dir, now.Add(-dotnetLeftoverAge), now.Add(-msbuildTempAge))
	return append(env, "TMPDIR="+dir, "TMP="+dir, "TEMP="+dir)
}

// sweepDotnetTemp removes per-command folders older than before and MSBuild's process folders older than
// msbuildBefore; anything else in dir is left alone.
func sweepDotnetTemp(dir string, before, msbuildBefore time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		cutoff := before
		switch {
		case !e.IsDir():
			continue
		case msbuildTemp.MatchString(e.Name()):
			cutoff = msbuildBefore
		case !dotnetLeftover.MatchString(e.Name()):
			continue
		}
		if info, err := e.Info(); err == nil && info.ModTime().Before(cutoff) {
			_ = safefs.RemoveAll(filepath.Join(dir, e.Name()))
		}
	}
}
