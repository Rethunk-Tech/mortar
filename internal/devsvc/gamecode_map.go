package devsvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Mapped is a type as C# and as IL, with Links[i] the C# line (from 0) that IL line i was compiled from, -1 for none.
type Mapped struct {
	CSharp string `json:"csharp"`
	IL     string `json:"il"`
	Links  []int  `json:"links"`
}

var sequencePoint = regexp.MustCompile(`^\s*// sequence point: \(line (\d+), col (\d+)\)`)

// maxLineShift bounds how far the PDB's line numbers may sit from the single type's C#; ILSpy writes the PDB's
// source as a project file, which differs from it by a line or so at the top.
const maxLineShift = 3

// DecompileMapped decompiles a type as C# and as IL and links each IL line to its C# line through the sequence points
// of a PDB ILSpy generates once per version of the assembly.
func (s *Service) DecompileMapped(ctx context.Context, gameID, dir, assembly, typ string) (Mapped, error) {
	if !typeName.MatchString(typ) {
		return Mapped{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a type name", typ))
	}
	path, err := s.assembly(gameID, dir, assembly)
	if err != nil {
		return Mapped{}, err
	}
	tool, ok := s.ilspy()
	if !ok {
		return Mapped{}, usererr.Wrap(usererr.NotFound, errors.New("ILSpy's command line (ilspycmd) is not installed"))
	}
	dll, err := s.withPDB(ctx, tool, path)
	if err != nil {
		return Mapped{}, err
	}
	cs, err := s.run(ctx, filepath.Dir(dll), tool, "-usepdb", "-t", typ, dll)
	if err != nil {
		return Mapped{}, err
	}
	out, err := s.run(ctx, filepath.Dir(dll), tool, "--il-sequence-points", "-usepdb", "-t", typ, dll)
	if err != nil {
		return Mapped{}, err
	}
	il := ilBlock(string(out), typ)
	return Mapped{CSharp: string(cs), IL: il, Links: linkLines(string(cs), il)}, nil
}

// withPDB is a copy of the assembly beside a PDB ILSpy generated for it, kept in the cache by the file's size and time.
func (s *Service) withPDB(ctx context.Context, tool, path string) (string, error) {
	st, err := fsx.Stat(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d", path, st.Size(), st.ModTime().UnixNano())))
	dir := filepath.Join(s.CacheDir, "game-code-pdb", hex.EncodeToString(sum[:8]))
	dll := filepath.Join(dir, filepath.Base(path))
	pdb := strings.TrimSuffix(dll, filepath.Ext(dll)) + ".pdb"
	if fsx.IsFile(dll) && fsx.IsFile(pdb) {
		return dll, nil
	}
	body, err := fsx.ReadFile(path)
	if err != nil {
		return "", err
	}
	if err := fsx.WriteFile(dll, body, 0o600); err != nil {
		return "", err
	}
	if _, err := s.run(ctx, dir, tool, "-genpdb", "-o", dir, dll); err != nil {
		return "", err
	}
	return dll, nil
}

// linkLines reads the IL's sequence points: each applies to the instructions after it, a blank line or another
// point ends it. The shift that lands most points on the start of a C# line's code is the one used.
func linkLines(cs, il string) []int {
	csLines := strings.Split(cs, "\n")
	ilLines := strings.Split(il, "\n")
	type point struct{ line, col int }
	raw := make([]point, len(ilLines))
	current := point{-1, 0}
	for i, l := range ilLines {
		if m := sequencePoint.FindStringSubmatch(l); m != nil {
			line, _ := strconv.Atoi(m[1])
			col, _ := strconv.Atoi(m[2])
			current = point{line - 1, col - 1}
		} else if strings.TrimSpace(l) == "" || !strings.HasPrefix(strings.TrimSpace(l), "IL_") {
			current = point{-1, 0}
		}
		raw[i] = current
	}
	best, bestHits := 0, -1
	for shift := -maxLineShift; shift <= maxLineShift; shift++ {
		hits := 0
		for _, p := range raw {
			if n := p.line + shift; p.line >= 0 && n >= 0 && n < len(csLines) {
				code := csLines[n]
				if p.col < len(code) && strings.TrimSpace(code[:p.col]) == "" && code[p.col] != ' ' && code[p.col] != '\t' {
					hits++
				}
			}
		}
		if hits > bestHits {
			best, bestHits = shift, hits
		}
	}
	links := make([]int, len(raw))
	for i, p := range raw {
		links[i] = -1
		if n := p.line + best; p.line >= 0 && n >= 0 && n < len(csLines) {
			links[i] = n
		}
	}
	return links
}
