package deploy

import (
	"cmp"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

const journalFile = "deploy.json"

// Place copies a loader's install-side files into the game folder. The game and its anti-cheat may write to the placed
// files, so they are copies, never links into the store.
type Place struct {
	Mode Mode
}

// Plan resolves the loaders' install-side plan files, whose Dst is relative to dir, the install folder.
func (Place) Plan(view View, dir string, files []launchplan.PlanFile) (Plan, error) {
	byDst := map[string]Op{}
	var owned []Owned
	for _, f := range files {
		base := dir
		if f.Root != "" {
			if base = view.Roots[f.Root]; base == "" || !filepath.IsAbs(base) {
				return Plan{}, fmt.Errorf("no %s folder to place %s in", f.Root, f.Dst)
			}
		}
		if !filepath.IsLocal(f.Dst) {
			return Plan{}, fmt.Errorf("%s leaves the %s folder", f.Dst, cmp.Or(f.Root, "install"))
		}
		dst := filepath.Join(base, f.Dst)
		// Folded so a later file wins over an earlier one that differs only in case, as the file system would.
		byDst[safefs.FoldCase(dst)] = Op{Src: f.Src, Dst: dst, WriteBack: f.Root != ""}
		if o, ok := ownedOf(base, f); ok && !slices.Contains(owned, o) {
			owned = append(owned, o)
		}
	}
	plan := Plan{Dir: dir, View: view, Owned: owned}
	for _, dst := range slices.Sorted(maps.Keys(byDst)) {
		plan.Ops = append(plan.Ops, byDst[dst])
	}
	return plan, nil
}

// ownedOf is the top folder of a path role's file, which its entry owns, paired with the profile's folder of that name
// (the plan file's Src ends with its Dst). A file at the role's root has none.
func ownedOf(base string, f launchplan.PlanFile) (Owned, bool) {
	top, _, nested := strings.Cut(filepath.ToSlash(f.Dst), "/")
	if f.Root == "" || !nested || !strings.HasSuffix(f.Src, f.Dst) {
		return Owned{}, false
	}
	profileRoot := strings.TrimSuffix(f.Src, f.Dst)
	return Owned{Dst: filepath.Join(base, top), Src: filepath.Join(profileRoot, top)}, true
}

func journalPath(dir string) string { return filepath.Join(dir, journalFile) }

// displacedPath is where the file an op set aside is kept. The Play's generation keeps a later Play's backup from landing
// on an earlier one at the same index.
func displacedPath(dir string, gen, i int) string {
	return filepath.Join(dir, "displaced", fmt.Sprint(gen), fmt.Sprint(i))
}

// HasJournal reports whether dir holds the record of a deploy not yet undone.
func HasJournal(dir string) bool {
	_, err := os.Stat(journalPath(dir))
	return err == nil
}

// Kept reports whether dir holds a journal a Play left in place on purpose.
func Kept(dir string) bool {
	var m Manifest
	found, err := datadir.ReadJournal(journalPath(dir), nil, &m)
	return err == nil && found && m.Keep
}

func persist(m Manifest) error {
	return datadir.WriteJSONIn(m.View.JournalDir, journalFile, m)
}

// ErrUnrecovered is Apply's refusal to deploy over the journal of a deploy that was never taken back: a second
// manifest would overwrite the first's record of the player's own files, and they would be lost.
var ErrUnrecovered = errors.New("an earlier deploy has not been taken back")

// Apply persists the manifest, then places the files.
func (p Place) Apply(ctx context.Context, q Plan) (Manifest, error) {
	prev, prevDst, reuse, err := p.carryover(q.View.JournalDir)
	if err != nil {
		return Manifest{}, err
	}
	if _, err := os.Lstat(journalPath(q.View.JournalDir)); err == nil && !reuse {
		if !prev.Keep {
			return Manifest{}, fmt.Errorf("%w: recover %s first", ErrUnrecovered, q.View.JournalDir)
		}
		// A Play that kept its files left this journal on purpose; this Play places everything, so it goes first.
		prev.Keep = false
		if err := p.Purge(ctx, prev); err != nil {
			return Manifest{}, err
		}
		prev, prevDst = Manifest{}, nil
	}
	// A plan that places only into a folder outside the install (a mods folder in Documents) needs no write access to it.
	if slices.ContainsFunc(q.Ops, func(o Op) bool { return safefs.UnderRoot(q.Dir, o.Dst) }) {
		if err := datadir.CheckWritable(q.Dir); err != nil {
			return Manifest{}, err
		}
	}
	m := Manifest{Dir: q.Dir, View: q.View, Owned: q.Owned, Existing: listOwned(q.Owned), Keep: p.Mode.Reuse != ReuseNone}
	var kept, carried []bool
	if reuse {
		// The player's own files are the ones the first Play found; the kept copies in the owned folders are Mortar's.
		m.Existing, m.Gen = prev.Existing, prev.Gen+1
		// A file the previous Play placed that this plan no longer wants leaves now; the record naming it survives the
		// removals, so a crash in between is recovered by the previous journal as it always was.
		if err := p.drop(prev, q); err != nil {
			return Manifest{}, err
		}
	}
	m.Ops, kept, carried, err = p.record(q, prev, prevDst)
	if err != nil {
		return Manifest{}, err
	}
	// The record comes first: from here a crash is recoverable.
	_ = safefs.Remove(logPath(q.View.JournalDir))
	if err := persist(m); err != nil {
		return Manifest{}, err
	}
	at("record")
	log, err := openLog(q.View.JournalDir)
	if err != nil {
		return Manifest{}, err
	}
	placed, runErr := p.placeAll(ctx, &m, log, kept, carried)
	commitErr := commitPlaced(&m, placed, log)
	if runErr != nil || commitErr != nil {
		return m, errors.Join(runErr, commitErr, log.close())
	}
	return m, errors.Join(log.close(), persist(m))
}

// record is the manifest's files: a file the previous Play left standing as it is, or one this Play puts down, with the
// source hash and sizes a later Play judges it unchanged by.
func (p Place) record(q Plan, prev Manifest, prevDst map[string]int) (ops []Placed, kept, carried []bool, err error) {
	kept = make([]bool, len(q.Ops))
	carried = make([]bool, len(q.Ops))
	for i, op := range q.Ops {
		prevOp, hadPrev := Placed{}, false
		if j, ok := prevDst[safefs.FoldCase(op.Dst)]; ok {
			prevOp, hadPrev = prev.Ops[j], true
		}
		if hadPrev && op.Src == prevOp.Src && p.Mode.Reuse.stands(prevOp) {
			prevOp.WriteBack = op.WriteBack
			ops = append(ops, prevOp)
			kept[i] = true
			continue
		}
		hash, size, mtime, err := sourceHash(op.Src, prevOp, hadPrev)
		if err != nil {
			return nil, nil, nil, err
		}
		pl := Placed{Dst: op.Dst, Src: op.Src, Hash: hash, WriteBack: op.WriteBack, SrcSize: size, SrcMTime: mtime}
		if hadPrev {
			// The bytes at Dst are the previous Play's copy, taken back before this one's goes down; the player's own
			// file, if one was set aside at the first Play, stays where it is.
			pl.Displaced, carried[i] = prevOp.Displaced, true
		} else if _, err := os.Lstat(op.Dst); err == nil {
			pl.Displaced = displacedPath(q.View.JournalDir, prev.Gen+1, i)
		}
		ops = append(ops, pl)
	}
	return ops, kept, carried, nil
}

// placeAll writes the files the record says this Play puts down, reporting progress once per file.
func (p Place) placeAll(ctx context.Context, m *Manifest, log *opLog, kept, carried []bool) ([]int, error) {
	var placed []int
	total := len(m.Ops)
	for i := range m.Ops {
		if kept[i] {
			total--
		}
	}
	if p.Mode.Progress != nil {
		p.Mode.Progress(0, total)
	}
	for i := range m.Ops {
		if err := ctx.Err(); err != nil {
			return placed, err
		}
		if kept[i] {
			continue
		}
		if err := p.put(m, i, log, carried[i]); err != nil {
			return placed, err
		}
		placed = append(placed, i)
		if p.Mode.Progress != nil {
			p.Mode.Progress(len(placed), total)
		}
	}
	return placed, nil
}

// carryover is the previous Play's record when this Play may build on it: the files it left in place, by folded
// destination, and whether the journal stays for the files this Play keeps. A journal that does not keep its files is
// returned too, for the Play that has to take it back before placing anything.
func (p Place) carryover(dir string) (Manifest, map[string]int, bool, error) {
	prev, byDst, found, err := previous(dir)
	if err != nil || !found {
		return Manifest{}, nil, false, err
	}
	if p.Mode.Reuse == ReuseNone || !prev.Keep {
		return prev, nil, false, nil
	}
	return prev, byDst, true, nil
}

// commitPlaced flushes the files p.put copied, once per file system they landed on, then marks each done in the log. A
// failed flush marks none.
func commitPlaced(m *Manifest, placed []int, log *opLog) error {
	type group struct{ files, dirs []string }
	groups := map[string]*group{}
	var order []string
	for _, i := range placed {
		dst := m.Ops[i].Dst
		dir := filepath.Dir(dst)
		key := dir
		if dev, err := safefs.Device(dir); err == nil {
			key = fmt.Sprint(dev)
		}
		g := groups[key]
		if g == nil {
			g = &group{}
			groups[key] = g
			order = append(order, key)
		}
		g.files = append(g.files, dst)
		if !slices.Contains(g.dirs, dir) {
			g.dirs = append(g.dirs, dir)
		}
	}
	for _, k := range order {
		g := groups[k]
		if err := safefs.Flush(g.dirs[0], g.files, g.dirs); err != nil {
			return err
		}
	}
	at("flushed")
	for _, i := range placed {
		m.Ops[i].Done = true
		recordPlaced(&m.Ops[i])
		if err := log.add(step{P: &i}, false); err != nil {
			return err
		}
		at("logged:" + fmt.Sprint(i))
	}
	return nil
}

// recordPlaced fills in the size and time the placed file now has, which the next Play judges "unchanged" by. A file
// that is not a regular one (a symlink, a clone that some file system reports oddly) records none, so it is placed again.
func recordPlaced(pl *Placed) {
	st, err := os.Lstat(pl.Dst)
	if err != nil {
		return
	}
	if !st.Mode().IsRegular() {
		pl.Size, pl.MTime = -1, 0
		return
	}
	pl.Size, pl.MTime = st.Size(), st.ModTime().UnixNano()
}

func (p Place) put(m *Manifest, i int, log *opLog, carry bool) error {
	pl := &m.Ops[i]
	n := fmt.Sprint(i)
	switch {
	case carry:
		// The previous Play's copy goes before this one's arrives; a backup the first Play took stays where it is.
		if err := os.MkdirAll(filepath.Dir(pl.Dst), 0o700); err != nil {
			return err
		}
		if err := safefs.Remove(pl.Dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		at("replaced:" + n)
	case pl.Displaced != "":
		if _, err := os.Lstat(pl.Dst); err == nil {
			if err := os.MkdirAll(filepath.Dir(pl.Displaced), 0o700); err != nil {
				return err
			}
			// Over a backup an earlier Play left: what stands at Dst now is newer than it, and is what a Purge gives back.
			if err := move(pl.Dst, pl.Displaced); err != nil {
				return err
			}
			at("displaced:" + n)
		}
	}
	if err := mkdirTracked(m, filepath.Dir(pl.Dst), log, n); err != nil {
		return err
	}
	at("before-copy:" + n)
	if err := datadir.CopyFileNoSync(pl.Src, pl.Dst); err != nil {
		return err
	}
	at("copied:" + n)
	return nil
}

// mkdirTracked creates dir and records the folders it had to make. Each is on the log, flushed, before it exists, so a
// crash cannot strand a folder Mortar made.
func mkdirTracked(m *Manifest, dir string, log *opLog, n string) error {
	var missing []string
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(d); err == nil || filepath.Dir(d) == d {
			break
		}
		missing = append(missing, d)
	}
	if len(missing) == 0 {
		return nil
	}
	for _, d := range missing {
		if err := log.add(step{M: d}, true); err != nil {
			return err
		}
	}
	at("mkdir-logged:" + n)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	m.Created = append(m.Created, missing...)
	at("mkdir:" + n)
	return nil
}

// move renames, falling back to copy and remove across file systems.
func move(src, dst string) error {
	if err := safefs.Rename(src, dst); err == nil {
		return nil
	}
	if err := datadir.CopyFile(src, dst); err != nil {
		return err
	}
	return safefs.Remove(src)
}

// Purge removes the placed files and restores what they displaced, then drops the journal.
func (p Place) Purge(ctx context.Context, m Manifest) error {
	var log *opLog
	if m.View.JournalDir != "" {
		var err error
		if log, err = openLog(m.View.JournalDir); err != nil {
			return err
		}
	}
	if m.Keep {
		return p.keep(ctx, &m, log)
	}
	if err := undoOps(ctx, &m, log); err != nil {
		return errors.Join(err, log.close())
	}
	at("ops-undone")
	if err := adopt(&m, log); err != nil {
		return errors.Join(err, log.close())
	}
	at("adopted")
	// Deepest first: a folder made for one file may hold a sibling's folder made later.
	for _, d := range slices.SortedFunc(slices.Values(m.Created), func(a, b string) int { return len(b) - len(a) }) {
		_ = safefs.Remove(d)
		at("folder:" + d)
	}
	err := log.close()
	at("journal")
	return errors.Join(err, saveNotes(m.View.JournalDir, m.notes), safefs.RemoveAll(m.View.JournalDir))
}

// drop takes back the files the previous Play placed that this plan no longer wants, before this Play's record replaces
// the previous one.
func (p Place) drop(prev Manifest, q Plan) error {
	wanted := make(map[string]bool, len(q.Ops))
	for _, op := range q.Ops {
		wanted[safefs.FoldCase(op.Dst)] = true
	}
	for i, o := range prev.Ops {
		if o.Undone || wanted[safefs.FoldCase(o.Dst)] {
			continue
		}
		if err := undoDst(&prev, o, i, nil); err != nil {
			return err
		}
	}
	return nil
}

// keep is Purge for a Play that leaves its files in place: a file that still holds our bytes stays for the next Play,
// and one the game or a mod rewrote is taken back as always and dropped from the record. A journal with nothing left to
// keep goes, so the next Play starts clean.
func (p Place) keep(ctx context.Context, m *Manifest, log *opLog) error {
	var kept []Placed
	var errs []error
	for i, o := range m.Ops {
		if err := ctx.Err(); err != nil {
			return errors.Join(err, log.close())
		}
		if o.Undone {
			continue
		}
		if holdsOurBytes(o) {
			kept = append(kept, o)
			continue
		}
		if err := undoDst(m, o, i, log); err != nil {
			return errors.Join(err, log.close())
		}
		m.Ops[i].Undone = true
		errs = append(errs, log.add(step{U: &i}, false))
	}
	if len(kept) == 0 {
		// Nothing left to keep: the journal goes with the last file, and the next Play starts clean.
		errs = append(errs, log.close())
		at("journal")
		return errors.Join(errors.Join(errs...), saveNotes(m.View.JournalDir, m.notes), safefs.RemoveAll(m.View.JournalDir))
	}
	m.Ops = kept
	if err := adopt(m, log); err != nil {
		return errors.Join(errors.Join(errs...), err, log.close())
	}
	errs = append(errs, log.close())
	// The keep log's indices name the pre-shortened ops; leaving it would mark the wrong files undone on the next Play.
	_ = safefs.Remove(logPath(m.View.JournalDir))
	errs = append(errs, persist(*m))
	at("kept")
	return errors.Join(errs...)
}

// holdsOurBytes reports whether the file at Dst is still the one the record placed, which the next Play may keep.
func holdsOurBytes(o Placed) bool {
	if !o.Done {
		return false
	}
	st, err := os.Lstat(o.Dst)
	if err != nil || !st.Mode().IsRegular() || st.Size() != o.Size || st.ModTime().UnixNano() != o.MTime {
		return false
	}
	return true
}

func undoOps(ctx context.Context, m *Manifest, log *opLog) error {
	for i := len(m.Ops) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		o := m.Ops[i]
		if o.Undone {
			continue
		}
		if err := undoDst(m, o, i, log); err != nil {
			return err
		}
		// A displaced file with no backup left was never moved or is already back: Dst is the player's own.
		at(fmt.Sprintf("undone-file:%d", i))
		m.Ops[i].Undone = true
		if err := log.add(step{U: &i}, false); err != nil {
			return err
		}
		at(fmt.Sprintf("undone-logged:%d", i))
	}
	return nil
}

// undoDst restores the destination of one placed file: the displaced original goes back, or a file still holding our
// content is removed.
func undoDst(m *Manifest, o Placed, i int, log *opLog) error {
	// A crashed copy leaves its temp file; the name is Mortar's, so it goes whatever else is known of the operation.
	if err := safefs.Remove(tmpName(o.Dst)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	_, backupErr := os.Lstat(o.Displaced)
	switch {
	case o.Displaced != "" && backupErr == nil:
		// With the player's file set aside, whatever is at Dst is ours, even if the game wrote to it.
		if err := writeBackOrRescue(m, o, i, log); err != nil {
			return err
		}
		if err := safefs.Remove(o.Dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return move(o.Displaced, o.Dst)
	case o.Displaced == "":
		return undoUndisplaced(m, o, i, log)
	}
	return nil
}

// undoUndisplaced handles a placed file that displaced nothing, so Dst is ours only while it still holds our content. A
// file changed since holds the player's or a mod's own data (a script mod's settings): one placed from the profile's copy
// goes back into that copy, and any other stays.
func undoUndisplaced(m *Manifest, o Placed, i int, log *opLog) error {
	// A link a mod left where our file was holds none of our bytes, and hashing it would read, and a write-back copy,
	// whatever it points at. Removing the link loses nothing.
	if st, err := os.Lstat(o.Dst); err == nil && st.Mode()&fs.ModeSymlink != 0 {
		return safefs.Remove(o.Dst)
	}
	if h, err := safefs.HashFile(o.Dst, sha256.New()); err == nil {
		return undoChanged(m, o, i, log, h)
	}
	return nil
}

func undoChanged(m *Manifest, o Placed, i int, log *opLog, h string) error {
	switch {
	case h == o.Hash || !o.Done && emptyFile(o.Dst):
		return safefs.Remove(o.Dst)
	case !o.Done:
		// Copied unsynced and never logged: a mismatch is a torn copy or a file that is not ours, never the game's
		// change, so it goes neither into the profile nor away.
		return nil
	case o.WriteBack:
		wrote, err := writeBackFile(o, i, log)
		if err != nil {
			return err
		}
		if !wrote {
			m.notes = append(m.notes, Note{Path: o.Dst, Kind: NoteRescued})
			return nil
		}
		return safefs.Remove(o.Dst)
	}
	return nil
}

// emptyFile reports a file of no bytes, which is what a copy that never reached the disk leaves.
func emptyFile(p string) bool {
	st, err := os.Lstat(p)
	return err == nil && st.Mode().IsRegular() && st.Size() == 0
}

// writeBackOrRescue copies a changed Dst of a profile-copy file over its Src before Dst is removed and the displaced file
// returns. When the profile's folder is gone it keeps the bytes beside Dst under a rescue name, on the log first, so a
// changed file is never removed without a copy.
func writeBackOrRescue(m *Manifest, o Placed, i int, l *opLog) error {
	// A link a mod left in place of the file is not followed into the profile; removing it loses no bytes.
	if st, err := os.Lstat(o.Dst); !o.WriteBack || !o.Done || (err == nil && !st.Mode().IsRegular()) {
		return nil
	}
	h, err := safefs.HashFile(o.Dst, sha256.New())
	if errors.Is(err, fs.ErrNotExist) || (err == nil && h == o.Hash) {
		return nil
	}
	if err != nil {
		return err
	}
	// Dst holding the backup's bytes is the player's file half moved (a copy across file systems cut short), not ours.
	if b, err := safefs.HashFile(o.Displaced, sha256.New()); err == nil && b == h {
		return nil
	}
	wrote, err := writeBackFile(o, i, l)
	if err != nil || wrote {
		return err
	}
	rescue := RescueName(o.Dst)
	if err := l.add(step{R: rescue}, true); err != nil {
		return err
	}
	at(fmt.Sprintf("rescue-logged:%d", i))
	log.Printf("deploy: %s could not go back into its profile; its bytes are kept as %s", o.Dst, rescue)
	if err := safefs.Rename(o.Dst, rescue); err != nil {
		return err
	}
	at(fmt.Sprintf("rescued:%d", i))
	m.notes = append(m.notes, Note{Path: rescue, Kind: NoteRescued})
	return nil
}

// RescueName is where the bytes of a changed file are kept when its profile is gone.
func RescueName(dst string) string { return dst + ".mortar-rescued" }

// writeBackFile copies Dst over Src through a temp file and a rename, so the bytes are in both places until Dst is
// removed. It reports false, leaving Dst alone, when the profile's folder is gone.
func writeBackFile(o Placed, i int, log *opLog) (bool, error) {
	if _, err := os.Stat(filepath.Dir(o.Src)); errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := datadir.CopyFile(o.Dst, o.Src); err != nil {
		return false, err
	}
	at(fmt.Sprintf("written-back:%d", i))
	return true, log.add(step{W: &i}, false)
}

// Recover finishes an unfinished deploy found at journalDir: it does nothing while alive reports the game running,
// else it purges.
func (p Place) Recover(ctx context.Context, journalDir string, alive func() bool) error {
	var m Manifest
	found, err := datadir.ReadJournal(journalPath(journalDir), alive, &m)
	if err != nil {
		return fmt.Errorf("read the deploy journal: %w", err)
	}
	if !found {
		return nil
	}
	// The journal may have moved with the data folder since it was written.
	if old := m.View.JournalDir; old != journalDir {
		for i := range m.Ops {
			if rel, err := filepath.Rel(old, m.Ops[i].Displaced); old != "" && m.Ops[i].Displaced != "" && err == nil && filepath.IsLocal(rel) {
				m.Ops[i].Displaced = filepath.Join(journalDir, rel)
			}
		}
		m.View.JournalDir = journalDir
	}
	if err := replay(&m); err != nil {
		return fmt.Errorf("read the deploy log: %w", err)
	}
	if m.Keep {
		return nil
	}
	return p.Purge(ctx, m)
}

// DiscardKept takes back a journal a Play left in place, so a vanilla Play or a Play that places everything starts
// from the tree the first Play found.
func (p Place) DiscardKept(ctx context.Context, journalDir string) error {
	var m Manifest
	found, err := datadir.ReadJournal(journalPath(journalDir), nil, &m)
	if err != nil {
		return fmt.Errorf("read the deploy journal: %w", err)
	}
	if !found || !m.Keep {
		return nil
	}
	if old := m.View.JournalDir; old != journalDir {
		for i := range m.Ops {
			if rel, err := filepath.Rel(old, m.Ops[i].Displaced); old != "" && m.Ops[i].Displaced != "" && err == nil && filepath.IsLocal(rel) {
				m.Ops[i].Displaced = filepath.Join(journalDir, rel)
			}
		}
		m.View.JournalDir = journalDir
	}
	if err := replay(&m); err != nil {
		return fmt.Errorf("read the deploy log: %w", err)
	}
	m.Keep = false
	return p.Purge(ctx, m)
}
