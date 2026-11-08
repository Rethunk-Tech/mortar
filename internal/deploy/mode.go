package deploy

// Mode is how a placement puts its files down and what it reports while doing it.
type Mode struct {
	// Reuse decides which files a Play may leave in place; ReuseNone places everything, so the journal of the previous
	// Play is emptied by the next Purge as before.
	Reuse Reuse
	// Progress is called once per file, with the number done and the total.
	Progress func(done, total int)
}
