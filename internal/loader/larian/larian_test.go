package larian

import (
	"bytes"
	"encoding/binary"
	"math"
	"path/filepath"
	"strings"
	"testing"

	larianpak "github.com/Rethunk-Tech/go-larian"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/deploy"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
	"github.com/pierrec/lz4/v4"
)

const game = "baldurs-gate-3"

func meta(uuid, name string) string {
	return `<?xml version="1.0" encoding="UTF-8"?><save><region id="Config"><node id="root"><children><node id="ModuleInfo">` +
		`<attribute id="Folder" type="LSString" value="` + name + `"/><attribute id="Name" type="LSString" value="` + name + `"/>` +
		`<attribute id="UUID" type="FixedString" value="` + uuid + `"/><attribute id="Version64" type="int64" value="36028797018963968"/>` +
		`</node></children></node></region></save>`
}

// lspk is an uncompressed LSPK v18 package holding one file.
func lspk(t *testing.T, name, data string) []byte {
	t.Helper()
	const headerLen, entryLen = 40, 272
	entry := make([]byte, entryLen)
	copy(entry, name)
	binary.LittleEndian.PutUint32(entry[256:], headerLen)
	binary.LittleEndian.PutUint32(entry[264:], uint32(len(data)&math.MaxUint32))
	binary.LittleEndian.PutUint32(entry[268:], uint32(len(data)&math.MaxUint32))
	comp := make([]byte, lz4.CompressBlockBound(entryLen))
	n, err := lz4.CompressBlock(entry, comp, nil)
	if err != nil || n == 0 {
		t.Fatalf("compress: %d %v", n, err)
	}
	body := append(bytes.Repeat([]byte{0}, headerLen), data...)
	listOff := len(body)
	var head [8]byte
	binary.LittleEndian.PutUint32(head[:], 1)
	binary.LittleEndian.PutUint32(head[4:], uint32(n&math.MaxUint32))
	body = append(append(body, head[:]...), comp[:n]...)
	copy(body, "LSPK")
	binary.LittleEndian.PutUint32(body[4:], 18)
	binary.LittleEndian.PutUint64(body[8:], uint64(listOff))
	binary.LittleEndian.PutUint32(body[16:], uint32((len(body)-listOff)&math.MaxUint32))
	return body
}

func write(t *testing.T, path string, b []byte) {
	t.Helper()
	testfs.WriteAt(t, path, b)
}

func TestPrelaunchWritesModSettingsInProfileOrderAndTheDeployerRestoresTheOwn(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	profile := filepath.Join(root, "profile")
	uuidA, uuidB := "aaaaaaaa-0000-0000-0000-000000000001", "bbbbbbbb-0000-0000-0000-000000000002"
	for _, p := range []struct{ file, uuid, name string }{{"A.pak", uuidA, "ModA"}, {"B.pak", uuidB, "ModB"}} {
		pak := lspk(t, "Mods/"+p.name+"/meta.lsx", meta(p.uuid, p.name))
		write(t, filepath.Join(profile, "Mods", p.file), pak)
		write(t, filepath.Join(root, "store", p.name, p.file), pak)
	}
	write(t, filepath.Join(profile, "Mods", "override.pak"), lspk(t, "Public/x.txt", "x"))
	// B is earlier in the profile than A.
	view := loader.ProfileView{Game: game, Dir: profile, Enabled: []string{filepath.Join(root, "store", "ModB"), filepath.Join(root, "store", "ModA")}}
	if err := (extension{}).Prelaunch(view); err != nil {
		t.Fatal(err)
	}
	mods, err := larianpak.ParseModSettings(mustRead(t, generatedPath(profile)))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range mods {
		got = append(got, m.UUID)
	}
	if want := strings.Join([]string{larianpak.GustavDevUUID, uuidB, uuidA}, ","); strings.Join(got, ",") != want {
		t.Fatalf("order %v, want %s", got, want)
	}

	plan := launchplan.New(launchplan.ModeProfile)
	extension{}.Contribute(plan, view)
	if len(plan.Files) != 1 || plan.Files[0].Root != Role || plan.Files[0].Dst != "modsettings.lsx" {
		t.Fatalf("plan files %+v", plan.Files)
	}
	settingsDir := filepath.Join(root, "Public")
	own := filepath.Join(settingsDir, "modsettings.lsx")
	write(t, own, []byte("the player's own"))
	d := deploy.Place{}
	dv := deploy.View{JournalDir: filepath.Join(root, "journal"), Roots: map[string]string{Role: settingsDir}}
	dp, err := d.Plan(dv, filepath.Join(root, "install"), plan.Files)
	if err != nil {
		t.Fatal(err)
	}
	m, err := d.Apply(t.Context(), dp)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(mustRead(t, own), mustRead(t, generatedPath(profile))) {
		t.Fatal("the generated modsettings.lsx is not in place for the run")
	}
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, own)) != "the player's own" {
		t.Fatalf("the player's modsettings.lsx is not back: %q", mustRead(t, own))
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := safefs.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
