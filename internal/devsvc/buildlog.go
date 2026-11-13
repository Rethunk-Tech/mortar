package devsvc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-dotnet"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// keptBuildLines bounds each kept build log; a failing build's errors sit in its last lines.
const keptBuildLines = 2000

var unsafeLogName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// buildLogName is the file in BuildLogDir that holds the last build of the project in dir: its name, made safe for
// any file system, plus a hash of the folder so two projects with one name keep separate logs.
func buildLogName(name, dir string) string {
	sum := sha256.Sum256([]byte(dir))
	return unsafeLogName.ReplaceAllString(name, "_") + "-" + hex.EncodeToString(sum[:4]) + ".log"
}

// keepBuildLog logs one line per build and, when BuildLogDir is set, keeps the build's output for the diagnostics
// bundle, so a report of a failed build carries the error the player saw.
func (s *Service) keepBuildLog(p settings.DevProject, args []string, out []byte, err error, took time.Duration) {
	took = took.Round(time.Millisecond)
	problems := dotnet.Diagnostics(string(out))
	result := "succeeded"
	if err != nil {
		result = "failed: " + err.Error()
		first := ""
		for _, pr := range problems {
			if pr.Severity == "error" {
				first = pr.Code + " " + pr.Message
				break
			}
		}
		slog.Warn("devsvc: build failed", "project", p.Name, "args", strings.Join(args, " "), "took", took, "problems", len(problems), "err", err, "first", first)
	} else {
		slog.Info("devsvc: build", "project", p.Name, "args", strings.Join(args, " "), "took", took, "problems", len(problems))
	}
	if s.BuildLogDir == "" {
		return
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) > keptBuildLines {
		lines = lines[len(lines)-keptBuildLines:]
	}
	body := fmt.Sprintf("%s dotnet %s\nfolder: %s\nresult: %s after %s\n\n%s\n",
		time.Now().Format(time.RFC3339), strings.Join(args, " "), p.Dir, result, took, strings.Join(lines, "\n"))
	if err := os.MkdirAll(s.BuildLogDir, 0o700); err != nil {
		slog.Warn("devsvc: keeping the build log", "err", err)
		return
	}
	if err := safefs.WriteFileAtomic(filepath.Join(s.BuildLogDir, buildLogName(p.Name, p.Dir)), []byte(body), 0o600); err != nil {
		slog.Warn("devsvc: keeping the build log", "err", err)
	}
}
