package devsvc

import "testing"

func TestWithDependency(t *testing.T) {
	listed := "<Project>\n  <ItemGroup>\n    <ThunderstoreDependency Include=\"BepInEx-BepInExPack\" Version=\"5.4.2305\" />\n  </ItemGroup>\n</Project>\n"
	added := withDependency(listed, "Evaisa-LethalLib", "1.1.1")
	if want := "<Project>\n  <ItemGroup>\n    <ThunderstoreDependency Include=\"BepInEx-BepInExPack\" Version=\"5.4.2305\" />\n    <ThunderstoreDependency Include=\"Evaisa-LethalLib\" Version=\"1.1.1\" />\n  </ItemGroup>\n</Project>\n"; added != want {
		t.Fatalf("added:\n%s", added)
	}
	if bumped := withDependency(added, "Evaisa-LethalLib", "1.2.0"); bumped != withDependency(listed, "Evaisa-LethalLib", "1.2.0") {
		t.Fatalf("bumped:\n%s", bumped)
	}
	if fresh := withDependency("<Project>\n</Project>\n", "A-B", "1.0.0"); fresh != "<Project>\n  <ItemGroup>\n    <ThunderstoreDependency Include=\"A-B\" Version=\"1.0.0\" />\n  </ItemGroup>\n</Project>\n" {
		t.Fatalf("fresh:\n%s", fresh)
	}
}
