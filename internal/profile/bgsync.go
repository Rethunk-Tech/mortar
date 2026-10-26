package profile

import (
	"log"
	"sync"
)

// bgSync lays profiles' files out in the background after a change, one run per profile at a time: a request that
// arrives while a run is under way asks for one more run when it ends, however many arrive.
type bgSync struct {
	mu   sync.Mutex
	jobs map[string]*bgJob
	// locks serialise every layout of one profile, background or a launch's.
	locks map[string]*sync.Mutex
}

type bgJob struct {
	again bool
	done  chan struct{}
}

func (b *bgSync) lock(key string) *sync.Mutex {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.locks == nil {
		b.locks = map[string]*sync.Mutex{}
	}
	l := b.locks[key]
	if l == nil {
		l = &sync.Mutex{}
		b.locks[key] = l
	}
	return l
}

func (b *bgSync) request(key string, run func()) {
	b.mu.Lock()
	if j := b.jobs[key]; j != nil {
		j.again = true
		b.mu.Unlock()
		return
	}
	if b.jobs == nil {
		b.jobs = map[string]*bgJob{}
	}
	j := &bgJob{done: make(chan struct{})}
	b.jobs[key] = j
	b.mu.Unlock()
	go func() {
		for {
			run()
			b.mu.Lock()
			if !j.again {
				delete(b.jobs, key)
				b.mu.Unlock()
				close(j.done)
				return
			}
			j.again = false
			b.mu.Unlock()
		}
	}()
}

// wait returns once key's background run, if any, has finished.
func (b *bgSync) wait(key string) {
	b.mu.Lock()
	j := b.jobs[key]
	b.mu.Unlock()
	if j != nil {
		<-j.done
	}
}

// waitAll returns once no background run is left.
func (b *bgSync) waitAll() {
	b.mu.Lock()
	jobs := make([]*bgJob, 0, len(b.jobs))
	for _, j := range b.jobs {
		jobs = append(jobs, j)
	}
	b.mu.Unlock()
	for _, j := range jobs {
		<-j.done
	}
}

func syncKey(game, id string) string { return game + "/" + id }

// syncDropped lays the profile's files out again in the background after a mod was removed or switched on or off, so
// the profile matches the list now and not at the next launch, without holding up the caller. Skipped while the game
// runs the profile; a failure is logged, since the next launch syncs anyway.
func (s *Store) syncDropped(game, id string) {
	s.bg.request(syncKey(game, id), func() {
		s.mu.Lock()
		running := s.unlocked(game, id) != nil
		s.mu.Unlock()
		if running {
			return
		}
		if err := s.syncPackages(game, id, false); err != nil {
			log.Printf("profile %s/%s: laying out files after a change: %v", game, id, err)
		}
	})
}

// SyncPackages lays the enabled packages' files out in the profile (see syncPackages), after any background layout of
// it has finished, so a launch never starts on a half-laid profile.
func (s *Store) SyncPackages(gameID, id string) error {
	return s.SyncPackagesFor(gameID, id, false)
}

// SyncPackagesFor is SyncPackages for a launch: the dev helpers (Entry.DevOnly) are laid out only when testCopies.
func (s *Store) SyncPackagesFor(gameID, id string, testCopies bool) error {
	s.bg.wait(syncKey(gameID, id))
	return s.syncPackages(gameID, id, testCopies)
}

// WaitSyncs returns once every background layout has finished.
func (s *Store) WaitSyncs() { s.bg.waitAll() }
