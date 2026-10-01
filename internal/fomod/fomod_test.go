package fomod

import (
	"encoding/binary"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

func loadFixture(t *testing.T, name string) Config {
	t.Helper()
	var (
		b   []byte
		err error
	)
	switch name {
	case "choose-one.xml":
		b, err = fs.ReadFile(os.DirFS("testdata"), "choose-one.xml")
	case "flags.xml":
		b, err = fs.ReadFile(os.DirFS("testdata"), "flags.xml")
	case "required.xml":
		b, err = fs.ReadFile(os.DirFS("testdata"), "required.xml")
	case "folder.xml":
		b, err = fs.ReadFile(os.DirFS("testdata"), "folder.xml")
	default:
		t.Fatalf("unknown fixture %s", name)
	}
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
	ops := Resolve(cfg, Choices{"Options": {"Pack": {"Beta"}}}, EvalContext{})
	if len(ops) != 1 || ops[0].Source != "beta/manifest.json" {
		t.Fatalf("ops = %+v", ops)
	}
	if Match(cfg, Choices{"Options": {"Pack": {"Nope"}}}, EvalContext{}) {
		t.Fatal("vanished choice matched")
	}
	if !Match(cfg, Choices{"Options": {"Pack": {"Alpha"}}}, EvalContext{}) {
		t.Fatal("alpha should match")
	}
}

func TestFlagsConditionalInstall(t *testing.T) {
	cfg := loadFixture(t, "flags.xml")
	on := Resolve(cfg, Choices{"Pick": {"Mode": {"On"}}}, EvalContext{})
	if len(on) != 1 || on[0].Source != "extra.txt" {
		t.Fatalf("flag on: %+v", on)
	}
	off := Resolve(cfg, Choices{"Pick": {"Mode": {"Off"}}}, EvalContext{})
	if len(off) != 0 {
		t.Fatalf("flag off: %+v", off)
	}
}

func TestRequiredFiles(t *testing.T) {
	cfg := loadFixture(t, "required.xml")
	none := Resolve(cfg, Choices{"More": {"Extra": nil}}, EvalContext{})
	if len(none) != 1 || none[0].Source != "core.txt" {
		t.Fatalf("required only: %+v", none)
	}
	both := Resolve(cfg, Choices{"More": {"Extra": {"Addon"}}}, EvalContext{})
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
	if err := Apply(src, dst, Resolve(cfg, nil, EvalContext{})); err != nil {
		t.Fatal(err)
	}
	got, err := fs.ReadFile(os.DirFS(dst), filepath.ToSlash(filepath.Join("mod", "a.txt")))
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
	raw := make([]byte, 2+len(u)*2)
	raw[0], raw[1] = 0xFF, 0xFE
	for i, r := range u {
		binary.LittleEndian.PutUint16(raw[2+i*2:], r)
	}
	path := filepath.Join(dir, "ModuleConfig.xml")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
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
	if len(VisibleSteps(cfg, nil, EvalContext{})) != 0 {
		t.Fatal("missing file should hide the step")
	}
	files := FileIndex(func(name string) string {
		if name == "other.dll" {
			return FileActive
		}
		return FileMissing
	})
	if len(VisibleSteps(cfg, nil, EvalContext{Files: files})) != 1 {
		t.Fatal("present file should show the step")
	}
}

func TestGameDependencyUnknownVersionUnmet(t *testing.T) {
	cfg, err := Parse([]byte(`<config><moduleName>D</moduleName>
		<installSteps><installStep name="S"><visible>
			<gameDependency version="1.6.0"/>
		</visible><optionalFileGroups><group name="G" type="SelectAny">
			<plugins><plugin name="P"><files><file source="p.txt" destination="p.txt"/></files>
			<typeDescriptor><type name="Optional"/></typeDescriptor></plugin></plugins>
		</group></optionalFileGroups></installStep></installSteps></config>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(VisibleSteps(cfg, nil, EvalContext{})) != 0 {
		t.Fatal("unknown game version should hide the step")
	}
}

func TestGameDependencyMetWhenInstalledVersionSufficient(t *testing.T) {
	cfg, err := Parse([]byte(`<config><moduleName>D</moduleName>
		<installSteps><installStep name="S"><visible>
			<gameDependency version="1.6.0"/>
		</visible><optionalFileGroups><group name="G" type="SelectAny">
			<plugins><plugin name="P"><files><file source="p.txt" destination="p.txt"/></files>
			<typeDescriptor><type name="Optional"/></typeDescriptor></plugin></plugins>
		</group></optionalFileGroups></installStep></installSteps></config>`))
	if err != nil {
		t.Fatal(err)
	}
	ctx := EvalContext{GameVersion: "1.6.15"}
	if len(VisibleSteps(cfg, nil, ctx)) != 1 {
		t.Fatal("installed game version should satisfy minimum")
	}
	if len(VisibleSteps(cfg, nil, EvalContext{GameVersion: "1.5.0"})) != 0 {
		t.Fatal("older game version should not satisfy minimum")
	}
}
