package larian

import (
	"bytes"
	"encoding/binary"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pierrec/lz4/v4"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

const meta = `<?xml version="1.0" encoding="UTF-8"?>
<save><region id="Config"><node id="root"><children><node id="ModuleInfo">
<attribute id="Folder" type="LSString" value="ImpUI_26922ba9"/>
<attribute id="Name" type="LSString" value="ImpUI &amp; more"/>
<attribute id="UUID" type="FixedString" value="26922ba9-6018-5252-075d-7ff2ba6ed879"/>
<attribute id="Version64" type="int64" value="36028797018963970"/>
</node></children></node></region></save>`

// u32 is a test size as the pak's 32-bit field; every size here is far below the limit.
func u32(n int) uint32 { return uint32(n & math.MaxUint32) }

// writePak builds an LSPK v18 package holding files, the meta.lsx LZ4-compressed as the game packs it.
func writePak(t *testing.T, path string, files map[string]string) {
	t.Helper()
	var data bytes.Buffer
	data.WriteString("LSPK")
	data.Write(make([]byte, 36))
	var list []byte
	for name, body := range files {
		packed := make([]byte, lz4.CompressBlockBound(len(body)))
		n, err := lz4.CompressBlock([]byte(body), packed, nil)
		if err != nil {
			t.Fatal(err)
		}
		e := make([]byte, entrySize)
		copy(e, strings.ReplaceAll(name, "/", "\\"))
		binary.LittleEndian.PutUint32(e[256:], u32(data.Len()))
		e[263] = compressLZ4
		binary.LittleEndian.PutUint32(e[264:], u32(n))
		binary.LittleEndian.PutUint32(e[268:], u32(len(body)))
		data.Write(packed[:n])
		list = append(list, e...)
	}
	listAt := data.Len()
	packed := make([]byte, lz4.CompressBlockBound(len(list)))
	n, err := lz4.CompressBlock(list, packed, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = binary.Write(&data, binary.LittleEndian, [2]uint32{u32(len(files)), u32(n)})
	data.Write(packed[:n])
	out := data.Bytes()
	binary.LittleEndian.PutUint32(out[4:], pakVersion)
	binary.LittleEndian.PutUint32(out[8:], u32(listAt))
	testfs.WriteFile(t, filepath.Dir(path), filepath.Base(path), string(out))
}

func TestPrelaunchListsEachEnabledPakWithAModule(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePak(t, filepath.Join(dir, "Mods", "k1", "ImpUI.pak"), map[string]string{
		"Mods/ImpUI_26922ba9/meta.lsx": meta, "Public/ImpUI/x.lsf": "x",
	})
	writePak(t, filepath.Join(dir, "Mods", "k2", "Override.pak"), map[string]string{"Public/Game/a.lsf": "a"})
	if err := (Loader{}).Prelaunch(loader.ProfileView{Game: "baldurs-gate-3", Dir: dir}); err != nil {
		t.Fatal(err)
	}
	b, err := fsx.ReadFile(filepath.Join(dir, settingsFile))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	gx, imp := strings.Index(got, `value="GustavX"`), strings.Index(got, `value="26922ba9-6018-5252-075d-7ff2ba6ed879"`)
	if gx < 0 || imp < gx || strings.Count(got, "ModuleShortDesc\">") != 2 || !strings.Contains(got, `value="ImpUI &amp; more"`) ||
		!strings.Contains(got, `id="Version64" type="int64" value="36028797018963970"`) {
		t.Fatalf("modsettings.lsx:\n%s", got)
	}
}

func TestAFileThatIsNotAPakIsLeftOut(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testfs.WriteFile(t, filepath.Join(dir, "Mods", "k"), "bad.pak", "PK\x03\x04 a zip")
	if err := (Loader{}).Prelaunch(loader.ProfileView{Game: "baldurs-gate-3", Dir: dir}); err != nil {
		t.Fatal(err)
	}
	if b, _ := fsx.ReadFile(filepath.Join(dir, settingsFile)); strings.Count(string(b), "ModuleShortDesc\">") != 1 {
		t.Fatalf("modsettings.lsx:\n%s", b)
	}
}
