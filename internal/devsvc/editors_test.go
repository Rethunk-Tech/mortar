package devsvc

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func placeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestEditorsFindsEachInstalledVersionOnce(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, "Local")
	programs := filepath.Join(root, "Program Files")
	older := filepath.Join(programs, "JetBrains", "JetBrains Rider 2024.3", "bin", "rider64.exe")
	newer := filepath.Join(programs, "JetBrains", "JetBrains Rider 2025.1", "bin", "rider64.exe")
	code := filepath.Join(local, "Programs", "Microsoft VS Code", "bin", "code.cmd")
	for _, path := range []string{older, newer, code} {
		placeFile(t, path, "")
	}
	placeFile(t, filepath.Join(programs, "JetBrains", "JetBrains Rider 2025.1", "product-info.json"), `{"version":"2025.1.2"}`)
	env := map[string]string{"LOCALAPPDATA": local, "ProgramFiles": programs}
	host := editorHost{
		windows: true,
		getenv:  func(name string) string { return env[name] },
		lookPath: func(bin string) (string, error) {
			if bin == "code" {
				return code, nil
			}
			return "", errors.New("not on PATH")
		},
	}
	got := host.find()
	want := []Editor{
		{Name: "VS Code", Path: code},
		{Name: "Rider 2025.1.2", Path: newer},
		{Name: "Rider", Path: older},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseVSWhereListsEveryInstall(t *testing.T) {
	got := parseVSWhere([]byte(`[
		{"displayName":"Visual Studio Community 2022","productPath":"C:\\VS\\2022\\devenv.exe","catalog":{"productDisplayVersion":"17.12.3"}},
		{"displayName":"Visual Studio Professional 2019","productPath":"C:\\VS\\2019\\devenv.exe","catalog":{"productDisplayVersion":"16.11.40"}},
		{"displayName":"Build Tools","productPath":""}
	]`))
	want := []Editor{
		{Name: "Visual Studio Community 2022 17.12.3", Path: `C:\VS\2022\devenv.exe`},
		{Name: "Visual Studio Professional 2019 16.11.40", Path: `C:\VS\2019\devenv.exe`},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v", got)
	}
	if parseVSWhere([]byte("not json")) != nil {
		t.Fatal("broken output should list nothing")
	}
}

func TestCommandExe(t *testing.T) {
	for command, want := range map[string]string{
		`"C:\Program Files\Rider\rider64.exe" "%1"`: `C:\Program Files\Rider\rider64.exe`,
		`C:\Program Files\Rider\rider64.EXE "%1"`:   `C:\Program Files\Rider\rider64.EXE`,
		`rundll32 shell32,OpenAs`:                   "",
	} {
		if got := commandExe(command); got != want {
			t.Errorf("%s: got %q", command, got)
		}
	}
}

func TestEditorArguments(t *testing.T) {
	const (
		dir = "/mods/ShopDrop"
		sln = "/mods/ShopDrop/ShopDrop.slnx"
		src = "/mods/ShopDrop/Plugin.cs"
	)
	for program, want := range map[string]struct{ project, file []string }{
		"/usr/bin/code":      {[]string{dir}, []string{"--goto", src + ":7:2"}},
		"/opt/r/rider.sh":    {[]string{sln}, []string{"--line", "7", src}},
		"/vs/devenv.exe":     {[]string{sln}, []string{"/Edit", src}},
		"/usr/bin/monodevel": {[]string{sln}, []string{src}},
	} {
		if got := projectArgs(program, dir, sln); !slices.Equal(got, want.project) {
			t.Errorf("%s project: %v", program, got)
		}
		if got := fileArgs(program, src, 7, 2); !slices.Equal(got, want.file) {
			t.Errorf("%s file: %v", program, got)
		}
	}
	if got := projectArgs("/opt/r/rider.sh", dir, ""); !slices.Equal(got, []string{dir}) {
		t.Errorf("no solution: %v", got)
	}
}

func TestSolutionInNeedsExactlyOne(t *testing.T) {
	dir := t.TempDir()
	if got := solutionIn(dir); got != "" {
		t.Fatalf("none: %q", got)
	}
	placeFile(t, filepath.Join(dir, "A.slnx"), "")
	if got := solutionIn(dir); filepath.Base(got) != "A.slnx" {
		t.Fatalf("one: %q", got)
	}
	placeFile(t, filepath.Join(dir, "B.sln"), "")
	if got := solutionIn(dir); got != "" {
		t.Fatalf("two: %q", got)
	}
}

func TestAChosenEditorThatIsGoneSaysSo(t *testing.T) {
	store, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(t.TempDir(), "gone", "rider64.exe")
	if _, err := store.Update(func(s *settings.Settings) { s.CodeEditor = gone }); err != nil {
		t.Fatal(err)
	}
	s := &Service{Settings: settings.NewService(store)}
	err = s.openIn("unused", "unused", func(string) []string { return nil })
	if err == nil || !strings.Contains(err.Error(), "rider64.exe") {
		t.Fatalf("got %v", err)
	}
}
