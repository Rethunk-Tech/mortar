// Package symsrv names the function a native crash address lies in, for the modules whose public PDBs Unity's and
// Microsoft's symbol servers publish. A PDB is fetched only when a crash is being explained, cached by its identity,
// and read just far enough to map an address to its nearest public symbol.
package symsrv

import (
	"bytes"
	"context"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

const (
	// maxPDBBytes bounds one download; Unity's player PDBs are tens of megabytes.
	maxPDBBytes = 512 << 20
	// maxCacheBytes bounds the cache folder; the oldest PDBs go first.
	maxCacheBytes  = 2 << 30
	codeViewMagic  = 0x53445352 // "RSDS"
	debugCodeView  = 2
	debugEntrySize = 28
	missMarker     = "miss"
)

// The symbol servers, by who publishes the PDBs. Unity documents only the http address; https is tried first.
var (
	unityServers     = []string{"https://symbolserver.unity3d.com/", "http://symbolserver.unity3d.com/"}
	microsoftServers = []string{"https://msdl.microsoft.com/download/symbols/"}
)

// unityPrefixes are the PDB names Unity's server serves, by prefix.
var unityPrefixes = []string{"unityplayer_", "windowsplayer_", "baselib_"}

// microsoftNames are the PDB names Microsoft's server serves that a game's crash passes through.
var microsoftNames = []string{
	"ntdll.pdb", "kernelbase.pdb", "kernel32.pdb", "ucrtbase.pdb", "vcruntime140.amd64.pdb",
	"vcruntime140_1.amd64.pdb", "msvcp140.amd64.pdb", "user32.pdb", "win32u.pdb", "d3d11.pdb", "dxgi.pdb", "coreclr.pdb",
	"clrjit.pdb",
}

// pdbName is a PDB file name a server path and the cache can hold: no folder, no traversal.
var pdbName = regexp.MustCompile(`^[A-Za-z0-9_.-]+\.pdb$`)

// CodeView is a module's RSDS debug record: the PDB it was linked with and that PDB's identity.
type CodeView struct {
	PDB  string
	GUID [16]byte
	Age  uint32
}

// ParseCodeView reads an RSDS record: the magic, the GUID, the age and the PDB path. Only the path's base name is kept.
func ParseCodeView(b []byte) (CodeView, bool) {
	if len(b) < 25 || binary.LittleEndian.Uint32(b) != codeViewMagic {
		return CodeView{}, false
	}
	var cv CodeView
	copy(cv.GUID[:], b[4:20])
	cv.Age = binary.LittleEndian.Uint32(b[20:])
	path, _, _ := bytes.Cut(b[24:], []byte{0})
	name := string(path)
	cv.PDB = name[strings.LastIndexAny(name, `/\`)+1:]
	return cv, pdbName.MatchString(cv.PDB)
}

// PECodeView reads the CodeView record of the PE file at path, a module the crash names on disk.
func PECodeView(path string) (CodeView, bool) {
	f, err := pe.Open(path)
	if err != nil {
		return CodeView{}, false
	}
	defer func() { _ = f.Close() }()
	var dir pe.DataDirectory
	switch h := f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		dir = h.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_DEBUG]
	case *pe.OptionalHeader64:
		dir = h.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_DEBUG]
	}
	if dir.Size == 0 || dir.Size > 64*debugEntrySize {
		return CodeView{}, false
	}
	for _, s := range f.Sections {
		if dir.VirtualAddress < s.VirtualAddress || dir.VirtualAddress+dir.Size > s.VirtualAddress+s.Size {
			continue
		}
		entries := make([]byte, dir.Size)
		if _, err := s.ReadAt(entries, int64(dir.VirtualAddress-s.VirtualAddress)); err != nil {
			return CodeView{}, false
		}
		for e := entries; len(e) >= debugEntrySize; e = e[debugEntrySize:] {
			if binary.LittleEndian.Uint32(e[12:]) != debugCodeView {
				continue
			}
			raw := make([]byte, min(binary.LittleEndian.Uint32(e[16:]), 1024))
			if _, err := readAtFile(path, raw, int64(binary.LittleEndian.Uint32(e[24:]))); err != nil {
				return CodeView{}, false
			}
			return ParseCodeView(raw)
		}
	}
	return CodeView{}, false
}

func readAtFile(path string, b []byte, off int64) (int, error) {
	f, err := fsx.Open(path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	return f.ReadAt(b, off)
}

// Key is the folder a symbol server files the PDB under: the GUID as 32 upper-case hex digits, its first three fields
// little-endian as Windows stores them, then the age in hex.
func (cv CodeView) Key() string {
	g := cv.GUID
	return fmt.Sprintf("%08X%04X%04X%X%X", binary.LittleEndian.Uint32(g[0:]), binary.LittleEndian.Uint16(g[4:]),
		binary.LittleEndian.Uint16(g[6:]), g[8:16], cv.Age)
}

// URLs are where the servers that publish the PDB serve it, best first; none for a PDB no listed server publishes,
// such as a mod's or the game's own.
func URLs(cv CodeView) []string {
	if !pdbName.MatchString(cv.PDB) {
		return nil
	}
	name := strings.ToLower(cv.PDB)
	path := cv.PDB + "/" + cv.Key() + "/" + cv.PDB
	var servers []string
	switch {
	case slices.ContainsFunc(unityPrefixes, func(p string) bool { return strings.HasPrefix(name, p) }):
		servers = unityServers
	case slices.Contains(microsoftNames, name):
		servers = microsoftServers
	}
	out := make([]string, len(servers))
	for i, s := range servers {
		out[i] = s + path
	}
	return out
}

// Resolver fetches and caches PDBs under Dir. Online says whether to try the network; a PDB already cached is read
// either way.
type Resolver struct {
	HTTP   *http.Client
	Dir    string
	Online func() bool
	// Wait is how long Name waits for a download before answering without it; the download carries on, so the
	// next crash view has the name.
	Wait time.Duration

	mu       sync.Mutex
	inflight map[string]chan struct{}
}

// Name is the public symbol the address rva of the module cv identifies lies in and the offset into it, as
// "GfxDeviceD3D11::DrawBuffers". ok is false when no server publishes its PDB, the PDB is not cached and cannot be
// fetched now, or no symbol covers rva.
func (r *Resolver) Name(ctx context.Context, cv CodeView, rva uint64) (name string, offset uint64, ok bool) {
	urls := URLs(cv)
	if len(urls) == 0 || r.Dir == "" {
		return "", 0, false
	}
	dir := filepath.Join(r.Dir, cv.PDB, cv.Key())
	path := filepath.Join(dir, cv.PDB)
	if !fsx.IsFile(path) {
		if freshMiss(dir) || r.Online != nil && !r.Online() {
			return "", 0, false
		}
		select {
		case <-r.fetch(context.WithoutCancel(ctx), dir, path, urls):
		case <-time.After(r.Wait):
			return "", 0, false
		case <-ctx.Done():
			return "", 0, false
		}
	}
	f, err := fsx.Open(path)
	if err != nil {
		return "", 0, false
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return "", 0, false
	}
	now := time.Now()
	_ = os.Chtimes(path, now, now)
	sym, off, err := Lookup(f, info.Size(), rva)
	return sym, off, err == nil
}

// fetch downloads the PDB once however many callers wait for it, and closes the returned channel when done. ctx
// should not end with the caller who gave up waiting: the download carries on for the next one.
func (r *Resolver) fetch(ctx context.Context, dir, path string, urls []string) chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.inflight == nil {
		r.inflight = map[string]chan struct{}{}
	}
	if done, ok := r.inflight[path]; ok {
		return done
	}
	done := make(chan struct{})
	r.inflight[path] = done
	go func() {
		defer func() {
			r.mu.Lock()
			delete(r.inflight, path)
			r.mu.Unlock()
			close(done)
		}()
		r.download(ctx, dir, path, urls)
	}()
	return done
}

// partMaxAge is how long a .part may sit before prune treats it as abandoned (a download is capped at 10 minutes).
const partMaxAge = time.Hour

// missMaxAge is how long a miss marker stands before the server is asked again (a symbol server may publish later).
const missMaxAge = 7 * 24 * time.Hour

func freshMiss(dir string) bool {
	info, err := fsx.Stat(filepath.Join(dir, missMarker))
	return err == nil && time.Since(info.ModTime()) <= missMaxAge
}

var errMissing = errors.New("the symbol server has no such PDB")

func (r *Resolver) download(ctx context.Context, dir, path string, urls []string) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return
	}
	missing := 0
	for _, u := range urls {
		err := r.get(ctx, u, path)
		if err == nil {
			r.prune()
			return
		}
		if errors.Is(err, errMissing) {
			missing++
		}
	}
	if missing == len(urls) {
		_ = fsx.WriteFile(filepath.Join(dir, missMarker), nil, 0o600)
	}
}

func (r *Resolver) get(ctx context.Context, url, path string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := r.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return errMissing
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("%s: %s", url, resp.Status)
	case resp.ContentLength > maxPDBBytes:
		return fmt.Errorf("%s: %d bytes is over the cap", url, resp.ContentLength)
	}
	// Only the in-flight download of this path gets here, so a .part already present is a crash's leftover.
	_ = fsx.Remove(path + ".part")
	tmp, err := fsx.CreateExcl(path+".part", 0o600)
	if err != nil {
		return err
	}
	_, err = io.Copy(tmp, fsx.CapReader(resp.Body, maxPDBBytes))
	if errors.Is(err, fsx.ErrTooLarge) {
		err = fmt.Errorf("%s: over the %d-byte cap", url, maxPDBBytes)
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = fsx.Remove(path + ".part")
		return err
	}
	return fsx.Rename(path+".part", path)
}

// prune deletes the least recently read PDBs until the cache is under its cap.
func (r *Resolver) prune() {
	type entry struct {
		path string
		size int64
		used time.Time
	}
	var all []entry
	var total int64
	_ = filepath.WalkDir(r.Dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if strings.HasSuffix(p, ".pdb.part") {
			if info, err := d.Info(); err == nil && time.Since(info.ModTime()) > partMaxAge {
				_ = fsx.Remove(p)
			}
			return nil
		}
		if !strings.HasSuffix(p, ".pdb") {
			return nil
		}
		if info, err := d.Info(); err == nil {
			all = append(all, entry{p, info.Size(), info.ModTime()})
			total += info.Size()
		}
		return nil
	})
	r.pruneMisses()
	slices.SortFunc(all, func(a, b entry) int { return a.used.Compare(b.used) })
	for _, e := range all {
		if total <= maxCacheBytes {
			return
		}
		if fsx.Remove(e.path) == nil {
			total -= e.size
		}
	}
}

// pruneMisses removes expired miss markers, then the key and PDB folders left empty. An empty folder younger than
// partMaxAge may belong to a download that has not written yet, so it stays.
func (r *Resolver) pruneMisses() {
	pdbs, _ := os.ReadDir(r.Dir)
	for _, pdb := range pdbs {
		if !pdb.IsDir() {
			continue
		}
		pdbDir := filepath.Join(r.Dir, pdb.Name())
		keys, _ := os.ReadDir(pdbDir)
		for _, key := range keys {
			keyDir := filepath.Join(pdbDir, key.Name())
			if info, err := fsx.Stat(filepath.Join(keyDir, missMarker)); err == nil && time.Since(info.ModTime()) > missMaxAge {
				_ = fsx.Remove(filepath.Join(keyDir, missMarker))
			}
			if info, err := fsx.Stat(keyDir); err == nil && info.IsDir() && time.Since(info.ModTime()) > partMaxAge {
				_ = fsx.Remove(keyDir)
			}
		}
		_ = fsx.Remove(pdbDir)
	}
}
