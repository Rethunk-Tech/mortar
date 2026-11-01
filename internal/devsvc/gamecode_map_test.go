package devsvc

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// ILSpy's PDB numbers lines one below the single type's C#; each point covers the instructions after it.
func TestLinkLinesFollowsSequencePointsAndFindsTheShift(t *testing.T) {
	cs := "class T\n{\n\tvoid Start()\n\t{\n\t\ta = 1;\n\t\tb();\n\t}\n}"
	il := ".method Start\n{\n" +
		"\t// sequence point: (line 6, col 3) to (line 6, col 9) in -/T.cs\n" +
		"\tIL_0000: ldc.i4.1\n\tIL_0001: stfld a\n" +
		"\t// sequence point: (line 7, col 3) to (line 7, col 7) in -/T.cs\n" +
		"\tIL_0006: call b\n" +
		"\t// sequence point: hidden\n\tIL_000b: ret\n}"
	want := []int{-1, -1, 4, 4, 4, 5, 5, -1, -1, -1}
	if got := linkLines(cs, il); !slices.Equal(got, want) {
		t.Fatalf("links = %v, want %v", got, want)
	}
}

func TestWithPDBKeepsOnlyTheAssembliesNewestVersion(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	svc.Run = func(_ context.Context, _, _ string, a ...string) ([]byte, error) {
		dll := a[len(a)-1]
		return nil, os.WriteFile(dll[:len(dll)-len(".dll")]+".pdb", []byte("pdb"), 0o600)
	}
	dll := filepath.Join(t.TempDir(), "Assembly-CSharp.dll")
	for i := range 2 {
		if err := os.WriteFile(dll, []byte("dll"), 0o600); err != nil {
			t.Fatal(err)
		}
		at := time.Unix(1_000_000+int64(i), 0)
		if err := os.Chtimes(dll, at, at); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.withPDB(t.Context(), "ilspycmd", dll); err != nil {
			t.Fatal(err)
		}
	}
	versions, _ := filepath.Glob(filepath.Join(svc.CacheDir, "game-code-pdb", "*", "*"))
	if len(versions) != 1 {
		t.Fatalf("versions = %v", versions)
	}
}
