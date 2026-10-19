package queue

import (
	"slices"
	"strings"
)

// fetchReady reports whether the item can download without the player's click.
func (s *Service) fetchReady(it *Item) bool {
	return it.Package != "" || it.Repo != "" || s.d.Premium() || s.usable(it) || s.stored(it)
}

// pendingNeeds is the unfinished items of the same profile that the item requires, through their own requirements.
func (s *Service) pendingNeeds(root *Item) []*Item {
	want := map[string]bool{}
	var walk func(it *Item)
	walk = func(it *Item) {
		for _, n := range it.Needs {
			if want[n] {
				continue
			}
			want[n] = true
			for _, d := range s.items {
				if d != root && d.Package != "" && strings.EqualFold(d.Package, n) && d.Game == root.Game && d.Profile == root.Profile {
					walk(d)
				}
			}
		}
	}
	walk(root)
	var out []*Item
	for _, d := range s.items {
		if d != root && d.Package != "" && want[strings.ToLower(d.Package)] && d.Game == root.Game && d.Profile == root.Profile && !dismissable(d.State) && d.State != StateInstalling {
			out = append(out, d)
		}
	}
	return out
}

// Top is "Download now": the item goes to the front of the queue, after the pending items it requires, which move
// with it, and a paused one is resumed. When every download slot is busy, the last running downloads that are not
// among them pause to make room and start again by themselves once slots are free.
func (s *Service) Top(id string) {
	s.mu.Lock()
	it := s.find(id)
	if it == nil || !slices.Contains([]string{StateQueued, StatePaused, StateWaitingClick}, it.State) {
		s.mu.Unlock()
		return
	}
	group := append(s.pendingNeeds(it), it)
	rest := slices.DeleteFunc(slices.Clone(s.items), func(x *Item) bool { return slices.Contains(group, x) })
	s.items = append(slices.Clone(group), rest...)
	need := 0
	for _, g := range group {
		if g.State == StatePaused {
			g.State, g.AutoResume = StateQueued, false
		}
		if g.State == StateQueued && s.fetchReady(g) {
			need++
		}
	}
	over := need - (s.slots() - s.downloading())
	for i := len(s.items) - 1; i >= 0 && over > 0; i-- {
		if v := s.items[i]; v.State == StateDownloading && !slices.Contains(group, v) {
			s.pauseLocked(v, true)
			over--
		}
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}

// Move shifts an unfinished item up (delta below zero) or down the queue past the neighbour it swaps with.
func (s *Service) Move(id string, delta int) {
	s.mu.Lock()
	i := slices.IndexFunc(s.items, func(it *Item) bool { return it.ID == id })
	if i >= 0 && !finished(s.items[i].State) && delta != 0 {
		dir := 1
		if delta < 0 {
			dir = -1
		}
		for j := i + dir; j >= 0 && j < len(s.items); j += dir {
			if !finished(s.items[j].State) {
				s.items[i], s.items[j] = s.items[j], s.items[i]
				break
			}
		}
	}
	s.mu.Unlock()
	s.publish(true)
}
