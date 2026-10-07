// Package tidy collects what Mortar repaired by itself at startup, so the window can say so once.
package tidy

import (
	"cmp"
	"log"
	"sync"
)

// ReportEvent is emitted when the startup repairs have finished and the report holds something.
const ReportEvent = "tidy:report"

// Entry is one kind of repair in one place. Names are what was touched, when it has names.
type Entry struct {
	What  string   `json:"what"`
	Where string   `json:"where"`
	Count int      `json:"count"`
	Names []string `json:"names"`
}

// Summary is everything repaired, with Total the number of things tidied.
type Summary struct {
	Total   int     `json:"total"`
	Entries []Entry `json:"entries"`
}

// Collector gathers entries until it is taken; entries added afterwards start the next report. Once closed it
// only logs, so repairs made later in a session do not turn up in the next report.
type Collector struct {
	mu      sync.Mutex
	entries []Entry
	closed  bool
}

// Add records a repair. Count defaults to the number of names; an entry that touched nothing is dropped.
func (c *Collector) Add(what, where string, count int, names ...string) {
	count = cmp.Or(count, len(names))
	if count <= 0 {
		return
	}
	if names == nil {
		names = []string{}
	}
	log.Printf("tidied: %s (%s): %d %v", what, where, count, names)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	for i := range c.entries {
		if e := &c.entries[i]; e.What == what && e.Where == where {
			e.Count += count
			e.Names = append(e.Names, names...)
			return
		}
	}
	c.entries = append(c.entries, Entry{What: what, Where: where, Count: count, Names: names})
}

// Close stops the collector accepting entries.
func (c *Collector) Close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
}

// Pending is how many entries wait to be taken.
func (c *Collector) Pending() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// Take returns what was collected and starts an empty report.
func (c *Collector) Take() Summary {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := Summary{Entries: c.entries}
	if out.Entries == nil {
		out.Entries = []Entry{}
	}
	for _, e := range out.Entries {
		out.Total += e.Count
	}
	c.entries = nil
	return out
}

// Service hands the report to the window, which asks once it has loaded.
type Service struct {
	Report *Collector
}

// Take is the pending report, once: asking again returns nothing until more is repaired.
func (s *Service) Take() Summary { return s.Report.Take() }
