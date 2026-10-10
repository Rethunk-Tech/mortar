package profile

import (
	"os"
	"path/filepath"
	"testing"
)

// syncRig is a sims4 profile whose profile folder is dir, with install and sync helpers.
type syncRig struct {
	t   *testing.T
	e   env
	p   Profile
	dir string
}

func newSyncRig(t *testing.T) *syncRig {
	t.Helper()
	e := newEnv(t)
	p, err := e.Create(folderGame, "S")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := e.ProfileDir(folderGame, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	return &syncRig{t: t, e: e, p: p, dir: dir}
}

func (r *syncRig) install(src Source, files map[string]string) Profile {
	r.t.Helper()
	res, err := r.e.InstallSource(r.t.Context(), folderGame, r.p.ID, zipOf(r.t, "a.zip", files), src)
	if err != nil {
		r.t.Fatal(err)
	}
	return res.Profile
}

func (r *syncRig) sync() {
	r.t.Helper()
	if err := r.e.SyncPackages(folderGame, r.p.ID); err != nil {
		r.t.Fatal(err)
	}
}

func (r *syncRig) path(rel string) string { return filepath.Join(r.dir, filepath.FromSlash(rel)) }

func (r *syncRig) read(rel string) string {
	b, _ := os.ReadFile(r.path(rel))
	return string(b)
}

func (r *syncRig) write(rel, body string) {
	r.t.Helper()
	if err := os.MkdirAll(filepath.Dir(r.path(rel)), 0o750); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(r.path(rel), []byte(body), 0o600); err != nil {
		r.t.Fatal(err)
	}
}

func (r *syncRig) want(rel, body string) {
	r.t.Helper()
	if got := r.read(rel); got != body {
		r.t.Fatalf("%s = %q, want %q", rel, got, body)
	}
}

func (r *syncRig) absent(rel string) {
	r.t.Helper()
	if _, err := os.Lstat(r.path(rel)); err == nil {
		r.t.Fatalf("%s is there", rel)
	}
}

func TestAMissingProfileCopyIsReseededFromTheStore(t *testing.T) {
	t.Parallel()
	r := newSyncRig(t)
	r.install(cfSource(10), map[string]string{"mc.package": "store bytes"})
	r.sync()
	if err := os.Remove(r.path("Mods/mc.package")); err != nil {
		t.Fatal(err)
	}
	r.sync()
	r.want("Mods/mc.package", "store bytes")
}

func TestACopyThatDiffersFromTheStoreIsKeptWithoutAValidRecord(t *testing.T) {
	t.Parallel()
	r := newSyncRig(t)
	r.install(cfSource(10), map[string]string{"mc.package": "store bytes", "same.package": "same"})
	r.sync()
	r.write("Mods/mc.package", "player=2 and more")
	record := r.path(placedFile)
	for name, damage := range map[string]func(){
		"missing":    func() { _ = os.Remove(record) },
		"garbage":    func() { _ = os.WriteFile(record, []byte("{not json"), 0o600) },
		"a list":     func() { _ = os.WriteFile(record, []byte(`["Mods/mc.package"]`), 0o600) },
		"unreadable": func() { _ = os.Remove(record); _ = os.MkdirAll(record, 0o750) },
	} {
		damage()
		if st, err := os.Stat(record); err == nil && st.IsDir() {
			// A folder where the record goes cannot be read; the sync must still keep the copy, then the record is rewritten.
			_ = os.RemoveAll(record)
		}
		for i := range 3 {
			r.sync()
			if got := r.read("Mods/mc.package"); got != "player=2 and more" {
				t.Fatalf("%s record, sync %d: the copy is %q", name, i+1, got)
			}
		}
	}
	r.want("Mods/same.package", "same")
}

func TestARecordIsReadFromDiskOnEverySync(t *testing.T) {
	t.Parallel()
	r := newSyncRig(t)
	r.install(cfSource(10), map[string]string{"mc.package": "store bytes"})
	r.sync()
	r.write("Mods/mc.package", "player=2")
	r.sync()
	rec := readPlaced(r.dir)["Mods/mc.package"]
	if rec.Hash == "" || rec.Size < 0 {
		t.Fatalf("the record after the syncs = %+v", rec)
	}
	for range 3 {
		r.sync()
	}
	r.want("Mods/mc.package", "player=2")
}

func TestAnUpdateThatNoLongerShipsAChangedFileHoldsItUnderChanged(t *testing.T) {
	t.Parallel()
	r := newSyncRig(t)
	r.install(cfSource(10), map[string]string{"mc/mc.ts4script": "s", "mc/old.cfg": "default"})
	r.sync()
	r.write("Mods/mc/old.cfg", "player=2")
	r.install(cfSource(11).WithReplacing(10), map[string]string{"mc/mc.ts4script": "s2"})
	r.sync()
	r.absent("Mods/mc/old.cfg")
	r.want("changed/Mods/mc/old.cfg", "player=2")
}

func TestRollBackKeepsTheChangedCopyAndHoldsOrRestoresTheRest(t *testing.T) {
	t.Parallel()
	r := newSyncRig(t)
	r.install(cfSource(10), map[string]string{"mc/mc.ts4script": "s", "mc/old.cfg": "default"})
	r.sync()
	r.write("Mods/mc/old.cfg", "player=2")
	cur := r.install(cfSource(11).WithReplacing(10), map[string]string{"mc/mc.ts4script": "s2", "mc/new.cfg": "default new"})
	r.sync()
	// old.cfg left with v1; new.cfg is v2's.
	r.want("changed/Mods/mc/old.cfg", "player=2")
	r.write("Mods/mc/new.cfg", "player=3")
	back, err := r.e.RollBack(folderGame, r.p.ID, cur.Entries[0].Key)
	if err != nil {
		t.Fatal(err)
	}
	r.sync()
	// v1 ships old.cfg again: the changed copy is the one placed, and v2's new.cfg is held, not deleted.
	r.want("Mods/mc/old.cfg", "player=2")
	r.absent("Mods/mc/new.cfg")
	r.want("changed/Mods/mc/new.cfg", "player=3")
	r.want("Mods/mc/mc.ts4script", "s")
	if _, err := r.e.RollBack(folderGame, r.p.ID, back.Entries[0].Key); err != nil {
		t.Fatal(err)
	}
	r.sync()
	r.want("Mods/mc/new.cfg", "player=3")
}

func TestUninstallHoldsTheChangedCopyAndReinstallBringsItBack(t *testing.T) {
	t.Parallel()
	r := newSyncRig(t)
	cur := r.install(cfSource(10), map[string]string{"mc/mc.ts4script": "s", "mc/mc.cfg": "default"})
	r.sync()
	r.write("Mods/mc/mc.cfg", "player=2")
	var keys []string
	for _, e := range cur.Entries {
		keys = append(keys, e.Key)
	}
	if _, err := r.e.RemoveEntries(folderGame, r.p.ID, keys); err != nil {
		t.Fatal(err)
	}
	r.sync()
	r.absent("Mods/mc/mc.cfg")
	r.want("changed/Mods/mc/mc.cfg", "player=2")
	r.install(cfSource(10), map[string]string{"mc/mc.ts4script": "s", "mc/mc.cfg": "default"})
	r.sync()
	r.want("Mods/mc/mc.cfg", "player=2")
	r.absent("changed/Mods/mc/mc.cfg")
}

func TestABackupAndRestoreRoundTripsChangedCopiesThroughRepeatedSyncs(t *testing.T) {
	t.Parallel()
	r := newSyncRig(t)
	r.install(cfSource(10), map[string]string{"mc/mc.ts4script": "s", "mc/mc.cfg": "default"})
	r.sync()
	r.write("Mods/mc/mc.cfg", "player=3")
	r.write("Mods/mc/created.dat", "adopted")
	prof, files, err := r.e.Backup(folderGame, r.p.ID)
	if err != nil {
		t.Fatal(err)
	}
	dirs, err := r.e.BackupDirs(folderGame, prof, func(Entry) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	restored, _, err := r.e.RestoreBackup(t.Context(), folderGame, prof, files, dirs)
	if err != nil {
		t.Fatal(err)
	}
	rr := &syncRig{t: t, e: r.e, p: restored}
	rr.dir, _ = r.e.ProfileDir(folderGame, restored.ID)
	for range 3 {
		rr.sync()
		rr.want("Mods/mc/mc.cfg", "player=3")
		rr.want("Mods/mc/created.dat", "adopted")
		rr.want("Mods/mc/mc.ts4script", "s")
	}
}
