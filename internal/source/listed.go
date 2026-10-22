package source

import "sync"

// listedCap is how many dependency lists a ListedStore remembers.
const listedCap = 4096

// ListedStore holds the required dependencies of versions an update check read, keyed by project and version, so
// Dependencies answers without another request. Oldest keys drop once the cap is hit. The zero value is ready.
type ListedStore struct {
	mu    sync.Mutex
	m     map[VersionRef][]string
	order []VersionRef
}

// Put records k's dependencies.
func (s *ListedStore) Put(k VersionRef, v []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[VersionRef][]string{}
	}
	if _, ok := s.m[k]; ok {
		s.m[k] = v
		return
	}
	for len(s.m) >= listedCap {
		old := s.order[0]
		s.order = s.order[1:]
		delete(s.m, old)
	}
	s.m[k] = v
	s.order = append(s.order, k)
}

// Get is k's recorded dependencies.
func (s *ListedStore) Get(k VersionRef) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[k]
	return v, ok
}

func (s *ListedStore) len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.m)
}
