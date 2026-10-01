package tools

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/game"
)

type Store struct {
	root string
}

func Open() (*Store, error) {
	base, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(base, "tools")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

func (s *Store) path(gameID string) (string, error) {
	if !game.Valid(gameID) {
		return "", fmt.Errorf("unknown game %q", gameID)
	}
	return filepath.Join(s.root, gameID+".json"), nil
}

func (s *Store) load(game string) ([]Tool, error) {
	path, err := s.path(game)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Tool{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	if f.Tools == nil {
		return []Tool{}, nil
	}
	return f.Tools, nil
}

func (s *Store) save(game string, tools []Tool) error {
	path, err := s.path(game)
	if err != nil {
		return err
	}
	return datadir.WriteJSON(path, file{Tools: tools})
}

func newID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func (s *Store) List(game string) ([]Tool, error) {
	return s.load(game)
}

func (s *Store) Add(game string, t Tool) (Tool, error) {
	if err := validateTool(t); err != nil {
		return Tool{}, err
	}
	t.Arguments = slices.Clone(t.Arguments)
	id, err := newID()
	if err != nil {
		return Tool{}, err
	}
	t.ID = id
	tools, err := s.load(game)
	if err != nil {
		return Tool{}, err
	}
	tools = append(tools, t)
	if err := s.save(game, tools); err != nil {
		return Tool{}, err
	}
	return t, nil
}

func (s *Store) Update(game string, t Tool) error {
	if t.ID == "" {
		return fmt.Errorf("id is required")
	}
	if err := validateTool(t); err != nil {
		return err
	}
	t.Arguments = slices.Clone(t.Arguments)
	tools, err := s.load(game)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(tools, func(x Tool) bool { return x.ID == t.ID })
	if i < 0 {
		return fmt.Errorf("tool not found")
	}
	tools[i] = t
	return s.save(game, tools)
}

func (s *Store) Remove(game string, id string) error {
	tools, err := s.load(game)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(tools, func(x Tool) bool { return x.ID == id })
	if i < 0 {
		return fmt.Errorf("tool not found")
	}
	tools = slices.Delete(tools, i, i+1)
	return s.save(game, tools)
}

// NormalizeArguments copies args for storage as a JSON array.
func NormalizeArguments(args []string) []string {
	if args == nil {
		return []string{}
	}
	return slices.Clone(args)
}
