package msbuild

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

// A hand-edited project: a commented-out decoy, a property with a Condition, an item spread over lines, tabs.
const project = `<Project Sdk="Microsoft.NET.Sdk">
	<!-- <Version>9.9.9</Version> was the old one -->
	<PropertyGroup>
		<Version Condition="'$(CI)' == ''">1.2.3</Version>
		<Authors>A &amp; B %3B C</Authors>
		<Empty />
	</PropertyGroup>
	<ItemGroup>
		<!-- <ThunderstoreDependency Include="Old-Thing" Version="0.0.1" /> -->
		<ThunderstoreDependency
			Version="5.4.2305"
			Include="BepInEx-BepInExPack" />
		<ThunderstoreDependency Include="Evaisa-LethalLib" Version="1.1.1" /><!-- keep -->
	</ItemGroup>
</Project>
`

func TestReadsPastCommentsAndUnescapes(t *testing.T) {
	if v := Property(project, "Version"); v != "1.2.3" {
		t.Errorf("Version = %q", v)
	}
	if v := Property(project, "Authors"); v != "A & B ; C" {
		t.Errorf("Authors = %q", v)
	}
	items, err := Items(project, "ThunderstoreDependency")
	if err != nil || len(items) != 2 || items[0].Include != "BepInEx-BepInExPack" || items[0].Attrs["Version"] != "5.4.2305" {
		t.Fatalf("items = %+v, %v", items, err)
	}
}

func TestEditsTouchOnlyTheirElement(t *testing.T) {
	got, err := SetProperty(project, "Version", "2.0.0")
	if err != nil || got != strings.Replace(project, ">1.2.3<", ">2.0.0<", 1) {
		t.Fatalf("set existing:\n%s", got)
	}
	got, _ = SetProperty(project, "Empty", "x;y")
	if !strings.Contains(got, "\t\t<Empty>x%3By</Empty>\n") {
		t.Errorf("set self-closing:\n%s", got)
	}
	got, _ = SetProperty(project, "Company", "Me")
	if !strings.Contains(got, "\t\t<Empty />\n\t\t<Company>Me</Company>\n\t</PropertyGroup>") {
		t.Errorf("add missing:\n%s", got)
	}
	item := Item{Type: "ThunderstoreDependency", Include: "Evaisa-LethalLib", Attrs: map[string]string{"Version": "1.2.0"}}
	deps := []string{"ThunderstoreDependency", "ThunderstoreSoftDependency"}
	isDep := func(it Item) bool { return it.Type == deps[0] || it.Type == deps[1] }
	got, _ = SetItem(project, item, deps, isDep)
	if got != strings.Replace(project, `<ThunderstoreDependency Include="Evaisa-LethalLib" Version="1.1.1" />`, `<ThunderstoreDependency Include="Evaisa-LethalLib" Version="1.2.0" />`, 1) {
		t.Errorf("replace:\n%s", got)
	}
	soft := Item{Type: "ThunderstoreSoftDependency", Include: "AinaVT-LethalConfig", Attrs: map[string]string{"Version": "1.4.6"}}
	got, _ = SetItem(project, soft, deps, isDep)
	if !strings.Contains(got, "<!-- keep -->\n\t\t<ThunderstoreSoftDependency Include=\"AinaVT-LethalConfig\" Version=\"1.4.6\" />\n\t</ItemGroup>") {
		t.Errorf("insert:\n%s", got)
	}
	got, _ = Remove(project, func(e Elem, parent string) bool {
		return parent == "ItemGroup" && e.Attrs["Include"] == "BepInEx-BepInExPack"
	})
	if got != strings.Replace(project, "\t\t<ThunderstoreDependency\n\t\t\tVersion=\"5.4.2305\"\n\t\t\tInclude=\"BepInEx-BepInExPack\" />\n", "", 1) {
		t.Errorf("remove multi-line:\n%s", got)
	}
	got, _ = SetItem("<Project>\n</Project>\n", soft, deps, isDep)
	if got != "<Project>\n  <ItemGroup>\n    <ThunderstoreSoftDependency Include=\"AinaVT-LethalConfig\" Version=\"1.4.6\" />\n  </ItemGroup>\n</Project>\n" {
		t.Errorf("new group:\n%s", got)
	}
}

func TestSetAttrRewritesOnlyThatAttribute(t *testing.T) {
	slnx := "<Solution>\n  <Project Path='src/Old/Old.csproj' Id=\"1\" />\n  <Project Path=\"tools/T.csproj\"/>\n</Solution>\n"
	got, err := SetAttr(slnx, func(e Elem, _ string) bool { return e.Name == "Project" && e.Attrs["Path"] == "src/Old/Old.csproj" }, "Path", "src/New/New.csproj")
	if err != nil || got != strings.Replace(slnx, "Path='src/Old/Old.csproj'", `Path="src/New/New.csproj"`, 1) {
		t.Fatalf("existing:\n%s", got)
	}
	got, _ = SetAttr(slnx, func(e Elem, _ string) bool { return e.Attrs["Path"] == "tools/T.csproj" }, "Type", "x")
	if !strings.Contains(got, `<Project Path="tools/T.csproj" Type="x" />`) {
		t.Fatalf("added:\n%s", got)
	}
}

func TestNamesMatchWithoutCase(t *testing.T) {
	text := "<Project>\n  <PropertyGroup>\n    <nullable>enable</nullable>\n  </PropertyGroup>\n  <ItemGroup>\n    <packagereference Include=\"P\" />\n  </ItemGroup>\n</Project>\n"
	out, err := RemoveProperty(text, "Nullable")
	if err != nil || strings.Contains(out, "nullable") {
		t.Errorf("RemoveProperty left %q, %v", out, err)
	}
	if items, _ := Items(text, "PackageReference"); len(items) != 1 {
		t.Errorf("Items = %+v", items)
	}
}

func TestParseRefusesMismatchedEndTags(t *testing.T) {
	if _, err := Parse("<a><b></a></b>"); err == nil {
		t.Error("<a><b></a></b> parsed")
	}
	if _, err := Parse("<a><b/></a>"); err != nil {
		t.Error(err)
	}
}

func TestSetPropertyRefusesCharactersXMLCannotCarry(t *testing.T) {
	for _, v := range []string{"\x10", "a\x00b", "\xff"} {
		if _, err := SetProperty(project, "Version", v); err == nil {
			t.Errorf("%q accepted", v)
		}
	}
	if _, err := SetProperty(project, "Version", "a\tb\nc"); err != nil {
		t.Error(err)
	}
}

func FuzzSetPropertyParses(f *testing.F) {
	f.Add("Version", "2.0;x")
	f.Add("New", "\x10")
	f.Fuzz(func(t *testing.T, name, value string) {
		// Callers pass fixed property names.
		if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(name) {
			return
		}
		out, err := SetProperty(project, name, value)
		if err != nil {
			return
		}
		if _, err := Parse(out); err != nil {
			t.Fatalf("SetProperty(%q, %q) wrote a file that does not parse: %v", name, value, err)
		}
	})
}

func TestAttributesAndItemsRefuseCharactersXMLCannotCarry(t *testing.T) {
	text := "<Project>\n  <ItemGroup>\n    <Reference Include=\"A\" />\n  </ItemGroup>\n</Project>\n"
	all := func(Elem, string) bool { return true }
	if _, err := SetAttr(text, all, "Private", "a\x10b"); !errors.Is(err, ErrBadChar) {
		t.Errorf("SetAttr: %v", err)
	}
	if _, err := SetItem(text, Item{Type: "Reference", Include: "B\x01"}, []string{"Reference"}, nil); !errors.Is(err, ErrBadChar) {
		t.Errorf("SetItem include: %v", err)
	}
	if _, err := SetItem(text, Item{Type: "Reference", Include: "B", Attrs: map[string]string{"HintPath": "x\x1f"}}, []string{"Reference"}, nil); !errors.Is(err, ErrBadChar) {
		t.Errorf("SetItem attr: %v", err)
	}
}
