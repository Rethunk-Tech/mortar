package fomod

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

func loadFixture(t *testing.T, name string) Config {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestChooseOne(t *testing.T) {
	cfg := loadFixture(t, "choose-one.xml")
	ops := Resolve(cfg, Choices{"Options": {"Pack": {"Beta"}}}, nil)
	if len(ops) != 1 || ops[0].Source != "beta/manifest.json" {
		t.Fatalf("ops = %+v", ops)
	}
	if Match(cfg, Choices{"Options": {"Pack": {"Nope"}}}, nil) {
		t.Fatal("vanished choice matched")
	}
	if !Match(cfg, Choices{"Options": {"Pack": {"Alpha"}}}, nil) {
		t.Fatal("alpha should match")
	}
}

func TestFlagsConditionalInstall(t *testing.T) {
	cfg := loadFixture(t, "flags.xml")
	on := Resolve(cfg, Choices{"Pick": {"Mode": {"On"}}}, nil)
	if len(on) != 1 || on[0].Source != "extra.txt" {
		t.Fatalf("flag on: %+v", on)
	}
	off := Resolve(cfg, Choices{"Pick": {"Mode": {"Off"}}}, nil)
	if len(off) != 0 {
		t.Fatalf("flag off: %+v", off)
	}
}

func TestRequiredFiles(t *testing.T) {
	cfg := loadFixture(t, "required.xml")
	none := Resolve(cfg, Choices{"More": {"Extra": nil}}, nil)
	if len(none) != 1 || none[0].Source != "core.txt" {
		t.Fatalf("required only: %+v", none)
	}
	both := Resolve(cfg, Choices{"More": {"Extra": {"Addon"}}}, nil)
	if len(both) != 2 {
		t.Fatalf("with addon: %+v", both)
	}
}

func TestFolderApply(t *testing.T) {
	cfg := loadFixture(t, "folder.xml")
	src := t.TempDir()
	dst := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "pack"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "pack", "a.txt"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Apply(src, dst, Resolve(cfg, nil, nil)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "mod", "a.txt"))
	if err != nil || string(got) != "ok" {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestFindConfigDepthAndUTF16(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "inner", "fomod")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	u := utf16.Encode([]rune(`<?xml version="1.0"?><config><moduleName>Wide</moduleName></config>`))
	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0xFE})
	for _, r := range u {
		buf.WriteByte(byte(r))
		buf.WriteByte(byte(r >> 8))
	}
	path := filepath.Join(dir, "ModuleConfig.xml")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := FindConfig(root)
	if err != nil || got != path {
		t.Fatalf("FindConfig = %q %v", got, err)
	}
	cfg, err := Load(path)
	if err != nil || cfg.ModuleName != "Wide" {
		t.Fatalf("Load = %+v %v", cfg, err)
	}
}

func TestFileDependencyUnmetWithoutIndex(t *testing.T) {
	cfg, err := Parse([]byte(`<config><moduleName>D</moduleName>
		<installSteps><installStep name="S"><visible>
			<fileDependency file="other.dll" state="Active"/>
		</visible><optionalFileGroups><group name="G" type="SelectAny">
			<plugins><plugin name="P"><files><file source="p.txt" destination="p.txt"/></files>
			<typeDescriptor><type name="Optional"/></typeDescriptor></plugin></plugins>
		</group></optionalFileGroups></installStep></installSteps></config>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(VisibleSteps(cfg, nil, nil)) != 0 {
		t.Fatal("missing file should hide the step")
	}
	files := FileIndex(func(name string) string {
		if name == "other.dll" {
			return FileActive
		}
		return FileMissing
	})
	if len(VisibleSteps(cfg, nil, files)) != 1 {
		t.Fatal("present file should show the step")
	}
}
