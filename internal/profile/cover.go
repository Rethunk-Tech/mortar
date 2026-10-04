package profile

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/modpic"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/game"
)

// CoverPath is the URL prefix a picked cover is served under, as <CoverPath><game>/<profile id>.
const CoverPath = "/profile-cover/"

// MaxCover caps a picked cover image, in bytes.
const MaxCover = 16 << 20

// coverFiles maps each sniffed image type Mortar keeps to the file name it takes in the profile folder. profile.json
// names that file, and only these names are ever opened, so an edited profile.json cannot point anywhere else.
var coverFiles = map[string]string{"image/png": "cover.png", "image/jpeg": "cover.jpg", "image/webp": "cover.webp"}

func coverType(name string) string {
	for typ, n := range coverFiles {
		if n == name {
			return typ
		}
	}
	return ""
}

// readCover reads the image at path, judging it by its content rather than its name.
func readCover(path string) ([]byte, string, error) {
	f, err := fsx.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, "", err
	}
	if !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf("%s is not a regular file", filepath.Base(path))
	}
	b, err := modpic.ReadCapped(f, MaxCover)
	if err != nil {
		return nil, "", err
	}
	name, ok := coverFiles[http.DetectContentType(b)]
	if !ok {
		return nil, "", fmt.Errorf("%s is not a PNG, JPEG or WebP image", filepath.Base(path))
	}
	return b, name, nil
}

func removeCover(dir, name string) error {
	if coverType(name) == "" {
		return nil
	}
	if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

var coverAfterWrite func(dir string)

// SetCover copies the image at path into the profile folder as its cover; the original is never referenced again.
func (s *Store) SetCover(game, id, path string) (Profile, error) {
	b, name, err := readCover(path)
	if err != nil {
		return Profile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	if err := datadir.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
		return Profile{}, err
	}
	if coverAfterWrite != nil {
		coverAfterWrite(dir)
	}
	old := p.Cover
	p.Cover = name
	p.Updated = time.Now().UTC().Truncate(time.Second)
	if err := writeProfile(dir, p); err != nil {
		return Profile{}, err
	}
	if old != name {
		_ = removeCover(dir, old)
	}
	return p, nil
}

// ClearCover drops the picked cover, so the hero goes back to the automatic one.
func (s *Store) ClearCover(game, id string) (Profile, error) {
	return s.update(game, id, func(p *Profile, dir string) error {
		if err := removeCover(dir, p.Cover); err != nil {
			return err
		}
		p.Cover = ""
		return nil
	})
}

// Covers lists the hero images to try, in order: the picked cover, the Nexus picture of the most-endorsed mod, then
// Steam's hero art for the game. The frontend moves down the list when one fails to load, and shows a solid tone
// after the last.
func Covers(gameID string, p Profile) []string {
	var out []string
	if coverType(p.Cover) != "" {
		out = append(out, CoverPath+gameID+"/"+p.ID+"?v="+strconv.FormatInt(p.Updated.Unix(), 10))
	}
	best := -1
	picture := ""
	for _, e := range p.Entries {
		src := e.Source
		if src.Kind == KindNexus && strings.HasPrefix(src.Picture, "https://") && src.EndorsementCount > best {
			best, picture = src.EndorsementCount, src.Picture
		}
	}
	if picture != "" {
		if local := modpic.AssetURL(picture); local != "" {
			out = append(out, local)
		}
		out = append(out, picture)
	}
	if g := game.Find(gameID); g != nil {
		out = append(out, game.ArtURL(g.SteamAppID()))
	}
	return out
}

// Covers reads the profile and lists its hero images.
func (s *Store) Covers(gameID, id string) ([]string, error) {
	p, err := s.read(gameID, id)
	if err != nil {
		return nil, err
	}
	return Covers(gameID, p), nil
}

// CoverMiddleware serves GET <CoverPath><game>/<profile id>: the profile's picked cover, from its own folder only.
// store returns nil until the profile store is open.
func CoverMiddleware(store func() *Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rest, ok := strings.CutPrefix(r.URL.Path, CoverPath)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			s := store()
			gameID, id, _ := strings.Cut(rest, "/")
			if r.Method != http.MethodGet || s == nil {
				http.NotFound(w, r)
				return
			}
			dir, err := s.profileDir(gameID, id)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			p, err := readAt(dir, id)
			typ := coverType(p.Cover)
			if err != nil || typ == "" {
				http.NotFound(w, r)
				return
			}
			f, err := fsx.Open(filepath.Join(dir, p.Cover))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer func() { _ = f.Close() }()
			info, err := f.Stat()
			if err != nil || !info.Mode().IsRegular() {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", typ)
			http.ServeContent(w, r, "", info.ModTime(), f)
		})
	}
}
