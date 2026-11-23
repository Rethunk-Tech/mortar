package installer

import (
	"strings"
	"testing"
)

func TestConvertToREDmodMovesTheArchivesBelowTheModule(t *testing.T) {
	t.Parallel()
	in := []File{{Src: "a.archive", Rel: "archive/pc/mod/a.archive"}, {Src: "x.xl", Rel: "Archive/PC/Mod/sub/x.xl"}}
	got, err := ConvertToREDmod(in, "My Mod", "1.2")
	if err != nil {
		t.Fatal(err)
	}
	if got.Files[0].Rel != "mods/My Mod/archives/a.archive" || got.Files[1].Rel != "mods/My Mod/archives/sub/x.xl" || got.Files[1].Src != "x.xl" {
		t.Fatalf("files = %+v", got.Files)
	}
	if got.InfoRel != "mods/My Mod/info.json" || got.Info != "{\n  \"name\": \"My Mod\",\n  \"version\": \"1.2\"\n}" {
		t.Fatalf("info = %q %q", got.InfoRel, got.Info)
	}
}

func TestConvertToREDmodRefusesWhatIsNotPurelyArchives(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		files []File
		want  string
	}{
		"other folder": {[]File{{Rel: "archive/pc/mod/a.archive"}, {Rel: "r6/scripts/a.reds"}}, "also has r6/scripts/a.reds"},
		"other file":   {[]File{{Rel: "archive/pc/mod/a.archive"}, {Rel: "archive/pc/mod/a.txt"}}, "also has archive/pc/mod/a.txt"},
		"only xl":      {[]File{{Rel: "archive/pc/mod/a.xl"}}, "no .archive files"},
	} {
		if _, err := ConvertToREDmod(c.files, "m", "1"); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestREDmodModuleNames(t *testing.T) {
	t.Parallel()
	if got, err := REDmodModuleName("  Cool.Mod: v2 \t  edit "); err != nil || got != "Cool_Mod_ v2 edit" {
		t.Fatalf("name = %q, %v", got, err)
	}
	if _, err := REDmodModuleName("  "); err == nil {
		t.Fatal("an empty name was taken")
	}
	if got := UniqueREDmodModuleName("A", []string{"a", "A 2"}); got != "A 3" {
		t.Fatalf("unique = %q", got)
	}
}
