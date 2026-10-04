// Package tools stores per-game external programs and launches them for a profile.
package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	gamepkg "github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/ids"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

func errToolNotFound() error {
	return usererr.Wrap(usererr.NotFound, fmt.Errorf("tool not found"))
}

type Store struct {
	root string
	mu   sync.Mutex
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

func (s *Store) load(game string) ([]Tool, error) {
	path, err := gamepkg.File(s.root, game)
	if err != nil {
		return nil, err
	}
	var f file
	found, err := datadir.ReadJSON(path, &f)
	if err != nil {
		return nil, err
	}
	if !found {
		return []Tool{}, nil
	}
	if f.Tools == nil {
		return []Tool{}, nil
	}
	return f.Tools, nil
}

func (s *Store) save(game string, tools []Tool) error {
	path, err := gamepkg.File(s.root, game)
	if err != nil {
		return err
	}
	return datadir.WriteJSON(path, file{Tools: tools})
}

func (s *Store) List(game string) ([]Tool, error) {
	return s.load(game)
}

func (s *Store) mutate(game string, fn func([]Tool) ([]Tool, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tools, err := s.load(game)
	if err != nil {
		return err
	}
	tools, err = fn(tools)
	if err != nil {
		return err
	}
	return s.save(game, tools)
}

func (s *Store) Add(game string, t Tool) (Tool, error) {
	t.Arguments = slices.Clone(t.Arguments)
	t.ID = ids.New()
	err := s.mutate(game, func(tools []Tool) ([]Tool, error) {
		return append(tools, t), nil
	})
	return t, err
}

func (s *Store) Update(game string, t Tool) error {
	if t.ID == "" {
		return fmt.Errorf("id is required")
	}
	t.Arguments = slices.Clone(t.Arguments)
	return s.mutate(game, func(tools []Tool) ([]Tool, error) {
		i := slices.IndexFunc(tools, func(x Tool) bool { return x.ID == t.ID })
		if i < 0 {
			return nil, errToolNotFound()
		}
		tools[i] = t
		return tools, nil
	})
}

func (s *Store) Remove(game, id string) error {
	return s.mutate(game, func(tools []Tool) ([]Tool, error) {
		i := slices.IndexFunc(tools, func(x Tool) bool { return x.ID == id })
		if i < 0 {
			return nil, errToolNotFound()
		}
		return slices.Delete(tools, i, i+1), nil
	})
}

// NormalizeArguments copies args for storage as a JSON array.
func NormalizeArguments(args []string) []string {
	if args == nil {
		return []string{}
	}
	return slices.Clone(args)
}
