package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// ServerDataFolder is the profile's folder for a dedicated server run: its config, saves and logs, and the Mods and
// ModConfig a launch places in it. It is never the client's data folder.
const ServerDataFolder = "ServerData"

// HasServer reports whether a profile of the game can run its dedicated server.
func HasServer(gameID string) bool {
	info, ok := components.Game(gameID)
	return ok && info.ProfileServer != nil
}

// ServerMode reports whether the profile runs the game's dedicated server; false when it cannot be read.
func (s *Store) ServerMode(gameID, id string) bool {
	p, err := s.read(gameID, id)
	return err == nil && p.Server
}

// ServerDataDir is the folder the server is told to keep its data in.
func (s *Store) ServerDataDir(gameID, id string) (string, error) {
	dir, err := s.ProfileDir(gameID, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ServerDataFolder), nil
}

// SetServerMode makes the profile run the dedicated server (or the client again). It never touches mods/.
func (s *Store) SetServerMode(gameID, id string, on bool) (Profile, error) {
	return s.update(gameID, id, func(p *Profile, _ string) error {
		p.Server = on
		return nil
	})
}

// MakeServer copies the profile, mods and mod config included, into a new profile that runs the dedicated server.
func (s *Store) MakeServer(gameID, id string) (Profile, error) {
	if !HasServer(gameID) {
		return Profile{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("%s has no dedicated server", gameID))
	}
	dup, err := s.Duplicate(gameID, id)
	if err != nil {
		return Profile{}, err
	}
	src, err := s.ProfileDir(gameID, id)
	if err != nil {
		return Profile{}, err
	}
	dst, err := s.ProfileDir(gameID, dup.ID)
	if err != nil {
		return Profile{}, err
	}
	if cfg := filepath.Join(src, ModConfigFolder); exists(cfg) {
		if err := datadir.CopyTree(cfg, filepath.Join(dst, ModConfigFolder)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Profile{}, err
		}
	}
	all, err := s.listOK(gameID)
	if err != nil {
		return Profile{}, err
	}
	const suffix = " server"
	base := []rune(dup.CopyOf)
	name := UniqueName(Names(all), string(base[:min(len(base), maxName-len(suffix))])+suffix)
	out, err := s.update(gameID, dup.ID, func(p *Profile, _ string) error {
		p.Server, p.Name = true, name
		return nil
	})
	if err == nil {
		s.listChanged(gameID)
	}
	return out, err
}
