package share

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func writeIn(t *testing.T, root, rel string, body []byte) {
	t.Helper()
	to := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func changedProfile(t *testing.T) (dir string, p profile.Profile) {
	t.Helper()
	dir = t.TempDir()
	writeIn(t, dir, "Mods/mc/mc.cfg", []byte("player=3"))
	writeIn(t, dir, "changed/Mods/old/o.cfg", []byte("held"))
	return dir, profile.Profile{Name: "P", Entries: []profile.Entry{{Key: "k", Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 70}}}}
}

// Changed copies ride in the payload under the switch that carries configs, so they follow the rules configs follow.
func TestChangedCopiesTravelUnderTheConfigSwitch(t *testing.T) {
	dir, p := changedProfile(t)
	for name, c := range map[string]struct {
		inc  Include
		want int
	}{"own computers": {OwnInclude(), 2}, "configs on": {Include{ConfigFiles: true}, 2}, "configs off": {Include{}, 0}} {
		var buf bytes.Buffer
		if _, err := Write(&buf, "sims4", p, filepath.Join(dir, "mods"), c.inc); err != nil {
			t.Fatal(err)
		}
		pv, err := ReadBytes(buf.Bytes())
		if err != nil || len(pv.ChangedFiles) != c.want {
			t.Fatalf("%s: %d changed files, %v", name, len(pv.ChangedFiles), err)
		}
	}
}

// Sync compares payloads by content: a changed copy that differs is a difference, as a changed config is.
func TestAChangedCopyThatDiffersIsADifferenceForSync(t *testing.T) {
	dir, p := changedProfile(t)
	read := func() Preview {
		var buf bytes.Buffer
		if _, err := Write(&buf, "sims4", p, filepath.Join(dir, "mods"), OwnInclude()); err != nil {
			t.Fatal(err)
		}
		pv, err := ReadBytes(buf.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		return pv
	}
	base := read()
	if !SameExceptArrival(base, read(), nil) {
		t.Fatal("two reads of one profile differ")
	}
	writeIn(t, dir, "Mods/mc/mc.cfg", []byte("player=4"))
	if SameExceptArrival(base, read(), nil) {
		t.Fatal("a rewritten changed copy was not a difference")
	}
}

// A copy over the per-file cap is listed as not sent, and the rest still travel.
func TestAChangedCopyOverTheCapIsListedAsNotSent(t *testing.T) {
	dir, p := changedProfile(t)
	writeIn(t, dir, "Mods/mc/world.dat", bytes.Repeat([]byte("x"), MaxConfigBytes+1))
	var buf bytes.Buffer
	skipped, err := Write(&buf, "sims4", p, filepath.Join(dir, "mods"), OwnInclude())
	if err != nil || !slices.Contains(skipped, "Mods/mc/world.dat") {
		t.Fatalf("skipped = %v, %v", skipped, err)
	}
	pv, err := ReadBytes(buf.Bytes())
	if err != nil || len(pv.ChangedFiles) != 2 {
		t.Fatalf("%d changed files, %v", len(pv.ChangedFiles), err)
	}
}

// A payload from another computer is untrusted: a changed entry with an unsafe path fails the whole file.
func TestAHostileChangedEntryFailsTheFile(t *testing.T) {
	_, p := changedProfile(t)
	for _, name := range []string{"changed/../x", "changed/Mods/../../x", "changed//abs", "changed/other/x", "changed/Mods/a\\b", "changed/Mods/CON", "changed/Mods/x.", "changed/profile.json"} {
		var good bytes.Buffer
		if _, err := Write(&good, "sims4", p, t.TempDir(), OwnInclude()); err != nil {
			t.Fatal(err)
		}
		zr, err := zip.NewReader(bytes.NewReader(good.Bytes()), int64(good.Len()))
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		zw := zip.NewWriter(&out)
		for _, f := range zr.File {
			rc, _ := f.Open()
			w, _ := zw.Create(f.Name)
			_, _ = w.Write(mustRead(t, rc))
		}
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte("bad"))
		_ = zw.Close()
		if _, err := ReadBytes(out.Bytes()); !errors.Is(err, ErrBadFile) || !strings.Contains(err.Error(), "unsafe") {
			t.Errorf("%q: %v", name, err)
		}
	}
}

func mustRead(t *testing.T, rc interface {
	Read([]byte) (int, error)
	Close() error
},
) []byte {
	t.Helper()
	defer func() { _ = rc.Close() }()
	var b bytes.Buffer
	if _, err := b.ReadFrom(rc); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
