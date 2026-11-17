package sharehost

import (
	"context"
	"sync"
	"time"
)

// MemStore is the in-memory store for tests and local dev.
type MemStore struct {
	mu    sync.Mutex
	codes map[string]*memCode
}

type memCode struct {
	meta     Meta
	versions map[int]Version
}

func NewMemStore() *MemStore {
	return &MemStore{codes: map[string]*memCode{}}
}

var _ Store = (*MemStore)(nil)

func (s *MemStore) Create(_ context.Context, code string, meta Meta, ver Version) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.codes[code]; ok {
		return ErrExists
	}
	s.codes[code] = &memCode{meta: meta, versions: map[int]Version{meta.Latest: ver}}
	return nil
}

func (s *MemStore) Meta(_ context.Context, code string) (Meta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[code]
	if !ok {
		return Meta{}, ErrNotFound
	}
	return c.meta, nil
}

func (s *MemStore) SaveMeta(_ context.Context, code string, meta Meta) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[code]
	if !ok {
		return ErrNotFound
	}
	c.meta = meta
	return nil
}

func (s *MemStore) SaveVersion(_ context.Context, code string, n int, ver Version) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[code]
	if !ok {
		return ErrNotFound
	}
	c.versions[n] = ver
	return nil
}

func (s *MemStore) LoadVersion(_ context.Context, code string, n int) (Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[code]
	if !ok {
		return Version{}, ErrNotFound
	}
	v, ok := c.versions[n]
	if !ok {
		return Version{}, ErrNotFound
	}
	return v, nil
}

func (s *MemStore) Sweep(_ context.Context, cutoff time.Time) (deleted, kept int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for code, c := range s.codes {
		if !ageRef(c.meta).After(cutoff) {
			delete(s.codes, code)
			deleted++
			continue
		}
		kept++
	}
	return deleted, kept, nil
}
