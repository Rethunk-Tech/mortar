package msbuild

import (
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
