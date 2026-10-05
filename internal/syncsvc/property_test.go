package syncsvc

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// A random run of two machines editing, scanning and answering offers must never lose or invent a change, and once
// the offers are answered both machines hold the newest edit. Seeds are fixed; MORTAR_SYNC_SEEDS raises the count.
func TestSyncTwoMachinesProperty(t *testing.T) {
	seeds := syncSeeds()
	for seed := range seeds {
		runSyncSequence(t, seed, 40, false)
	}
}

// The same with a sync tool that delivers each file of a profile on its own schedule.
func TestSyncLaggingDeliveryProperty(t *testing.T) {
	seeds := syncSeeds()
	for seed := range seeds {
		runSyncSequence(t, seed, 50, true)
	}
}

type syncRun struct {
	t        *testing.T
	seed     uint64
	step     int
	trace    []string
	written  map[string]bool
	edits    int
	machines [2]*machine
	services [2]*Service
	folder   string
	// folders holds each machine's own copy of the sync folder when delivery lags; otherwise both are folder.
	folders [2]string
	// seen is each folder file's last known content and tick is the logical clock stamping changed ones, so "newer"
	// does not depend on the filesystem's timestamp resolution.
	seen map[string]string
	tick int64
	// local marks files a machine wrote that no delivery has replaced yet; delivering over one is a clash the sync
	// tool settles by recency, which can drop a write whatever the app does.
	local   map[string]bool
	clashed bool
}

func (r *syncRun) fail(format string, args ...any) {
	r.t.Helper()
	r.t.Fatalf("seed %d step %d: %s\ntrace: %v", r.seed, r.step, fmt.Sprintf(format, args...), r.trace)
}

func (r *syncRun) scan(i int) []Offer {
	before := snapshot(r.machines[i])
	offers, err := r.services[i].Scan(context.Background())
	if err != nil {
		r.fail("scan %d: %v", i, err)
	}
	r.stamp()
	if after := snapshot(r.machines[i]); !maps.Equal(before, after) {
		r.fail("a scan changed machine %d's profiles: %v -> %v", i, before, after)
	}
	r.checkPayload()
	return offers
}

// stamp gives every sync-folder file that changed since the last look the next logical modification time.
func (r *syncRun) stamp() {
	for _, folder := range r.folders {
		for _, name := range []string{"main.mortar", "main.sync.json"} {
			path := folder + "/stardew/" + name
			b, err := fsx.ReadFile(path)
			if err != nil || r.seen[path] == string(b) {
				continue
			}
			r.seen[path] = string(b)
			r.local[path] = true
			r.tick++
			when := time.Unix(1_000_000+r.tick, 0)
			if err := os.Chtimes(path, when, when); err != nil {
				r.fail("%v", err)
			}
		}
	}
}

func snapshot(m *machine) map[string]string { return maps.Clone(m.profiles) }

// checkPayload: whatever the sync folder holds was written by a user edit, never made up.
func (r *syncRun) checkPayload() {
	for _, folder := range r.folders {
		b, err := fsx.ReadFile(folder + "/stardew/main.mortar")
		if err == nil && !r.written[string(b)] {
			r.fail("the sync folder holds %q, which no one wrote", b)
		}
	}
}

func (r *syncRun) edit(i int) {
	m := r.machines[i]
	id := "main"
	if i == 1 {
		id = "local-Main"
	}
	if _, ok := m.profiles[id]; !ok && (i == 1 || len(m.profiles) > 0) {
		return
	}
	r.edits++
	content := "e" + strconv.Itoa(r.edits)
	r.written[content] = true
	m.set(id, content)
	r.trace = append(r.trace, fmt.Sprintf("edit%d=%s", i, content))
}

func (r *syncRun) answer(i int, offers []Offer) {
	for _, o := range offers {
		choice := Theirs
		if o.Conflict {
			choice = r.winner(i)
		}
		before := snapshot(r.machines[i])
		if err := r.services[i].Resolve(context.Background(), "stardew", o.Remote, o.Revision, choice); err != nil {
			r.fail("resolve %d %s: %v", i, choice, err)
		}
		r.stamp()
		r.trace = append(r.trace, fmt.Sprintf("resolve%d:%s(conflict=%v A=%v B=%v)", i, choice, o.Conflict, r.machines[0].profiles, r.machines[1].profiles))
		if choice == Theirs {
			want, _ := fsx.ReadFile(r.folders[i] + "/stardew/" + o.Remote + ".mortar")
			for id, got := range r.machines[i].profiles {
				if got != string(want) && before[id] != got {
					r.fail("taking theirs left %q, the offered revision is %q", got, want)
				}
			}
		}
		r.checkPayload()
	}
}

// winner is the side holding the later edit, which a conflict keeps: discarding the newest change is the user's call,
// and the property is that nothing else loses it.
func (r *syncRun) winner(i int) string {
	mine := latest(r.machines[i])
	theirs := max(latest(r.machines[1-i]), r.sharedEdit())
	if mine >= theirs {
		return Mine
	}
	return Theirs
}

func latest(m *machine) int {
	best := -1
	for _, c := range m.profiles {
		if n, err := strconv.Atoi(c[1:]); err == nil && len(c) > 1 {
			best = max(best, n)
		}
	}
	return best
}

func runSyncSequence(t *testing.T, seed uint64, steps int, lagging bool) {
	t.Helper()
	rng := prng(seed*2654435761 + 1)
	folder := t.TempDir()
	folders := [2]string{folder, folder}
	if lagging {
		folders = [2]string{t.TempDir(), t.TempDir()}
	}
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := &syncRun{t: t, seed: seed, written: map[string]bool{}, seen: map[string]string{}, local: map[string]bool{}, folder: folder, folders: folders}
	r.services[0], r.machines[0] = newMachine(t, folders[0], "Desktop", &clock)
	r.services[1], r.machines[1] = newMachine(t, folders[1], "Laptop", &clock)
	r.edit(0)
	if len(r.machines[0].profiles) == 0 {
		r.machines[0].set("main", "e0")
		r.written["e0"] = true
	}
	for r.step = 0; r.step < steps; r.step++ {
		i := rng.Intn(2)
		if lagging && rng.Intn(3) == 0 {
			for _, name := range [][]string{{"main.mortar"}, {"main.sync.json"}, {"main.mortar", "main.sync.json"}}[rng.Intn(3)] {
				r.deliver(i, name)
			}
			continue
		}
		switch rng.Intn(5) {
		case 0:
			r.edit(i)
		case 1, 2:
			r.trace = append(r.trace, fmt.Sprintf("scan%d", i))
			r.scan(i)
		case 3:
			r.answer(i, r.scan(i))
		case 4:
			if i == 1 && rng.Intn(4) == 0 {
				for id := range r.machines[1].profiles {
					_ = r.machines[1].Delete("stardew", id)
				}
				r.trace = append(r.trace, "delete1")
			}
		}
	}
	r.settle()
}

// settle answers every offer until none is left, then both machines must agree on the newest edit still held.
func (r *syncRun) settle() {
	r.step = -1
	r.trace = append(r.trace, "SETTLE")
	expected := latest(r.machines[0])
	expected = max(expected, latest(r.machines[1]))
	expected = max(expected, r.sharedEdit())
	for round := range 12 {
		quiet := true
		before := r.folderState()
		if r.folders[0] != r.folders[1] {
			for _, name := range []string{"main.mortar", "main.sync.json"} {
				r.deliver(0, name)
				r.deliver(1, name)
			}
		}
		for i := range r.machines {
			offers := r.scan(i)
			r.trace = append(r.trace, fmt.Sprintf("s%d:%d", i, len(offers)))
			if len(offers) > 0 {
				quiet = false
				r.answer(i, offers)
			}
		}
		quiet = quiet && before == r.folderState()
		if quiet {
			break
		}
		if round == 11 {
			r.fail("offers never settled")
		}
	}
	for i := range r.machines {
		if offers := r.scan(i); len(offers) != 0 {
			r.fail("machine %d still has offers %+v", i, offers)
		}
	}
	a, b := r.machines[0].profiles, r.machines[1].profiles
	if len(a) == 0 || len(b) == 0 {
		return
	}
	if latest(r.machines[0]) != latest(r.machines[1]) {
		r.fail("machines diverged: %v vs %v; folders %s; stalled %v %v", a, b, r.folderState()+r.mtimes(), r.services[0].Stalled(), r.services[1].Stalled())
	}
	if expected > 0 && !r.clashed && latest(r.machines[0]) != expected {
		r.fail("the newest edit e%d was lost: %v vs %v", expected, a, b)
	}
}

// folderState is the sync folder's two files; a scan that pushes changes them.
func (r *syncRun) folderState() string {
	var out string
	for _, folder := range r.folders {
		a, _ := fsx.ReadFile(folder + "/stardew/main.mortar")
		b, _ := fsx.ReadFile(folder + "/stardew/main.sync.json")
		out += string(a) + "|" + string(b) + "||"
	}
	return out
}

// sharedEdit is the number of the edit the sync folder holds, -1 when none.
func (r *syncRun) sharedEdit() int {
	best := -1
	for _, folder := range r.folders {
		b, err := fsx.ReadFile(folder + "/stardew/main.mortar")
		if err != nil || len(b) < 2 {
			continue
		}
		if n, err := strconv.Atoi(string(b[1:])); err == nil {
			best = max(best, n)
		}
	}
	return best
}

// deliver copies one of the sync folder's files from machine from's copy to the other's, as a sync tool does, in
// whatever order it likes.
func (r *syncRun) deliver(from int, name string) {
	src, err := os.Stat(r.folders[from] + "/stardew/" + name)
	if err != nil {
		return
	}
	// Like a sync tool, the newer copy wins; an older or identical one is not delivered over it.
	if dst, err := os.Stat(r.folders[1-from] + "/stardew/" + name); err == nil && !dst.ModTime().Before(src.ModTime()) {
		return
	}
	raw, err := fsx.ReadFile(r.folders[from] + "/stardew/" + name)
	if err != nil {
		return
	}
	if r.local[r.folders[1-from]+"/stardew/"+name] && r.seen[r.folders[1-from]+"/stardew/"+name] != string(raw) {
		r.clashed = true
	}
	delete(r.local, r.folders[1-from]+"/stardew/"+name)
	to := r.folders[1-from] + "/stardew"
	if err := os.MkdirAll(to, 0o700); err != nil {
		r.fail("%v", err)
	}
	if err := fsx.WriteFile(to+"/"+name, raw, 0o600); err != nil {
		r.fail("%v", err)
	}
	if err := os.Chtimes(to+"/"+name, src.ModTime(), src.ModTime()); err != nil {
		r.fail("%v", err)
	}
	r.seen[to+"/"+name] = string(raw)
	r.trace = append(r.trace, fmt.Sprintf("deliver%d:%s", from, name))
}

func (r *syncRun) mtimes() string {
	var out string
	for _, folder := range r.folders {
		for _, name := range []string{"main.mortar", "main.sync.json"} {
			if st, err := os.Stat(folder + "/stardew/" + name); err == nil {
				out += fmt.Sprintf(" %s@%d", name, st.ModTime().UnixNano())
			}
		}
	}
	return out
}

// prng is a xorshift generator: a run is fully determined by its seed, on every Go version.
type prng uint64

func (p *prng) Intn(n int) int {
	x := uint64(*p)
	x ^= x << 13
	x ^= x >> 7
	x ^= x << 17
	*p = prng(x)
	return int(x>>33) % n
}

// syncSeeds is how many seeded runs a test makes: a few for the normal suite, MORTAR_SYNC_SEEDS for a long one.
func syncSeeds() uint64 {
	if v, err := strconv.ParseUint(os.Getenv("MORTAR_SYNC_SEEDS"), 10, 32); err == nil && v > 0 {
		return v
	}
	return 60
}
