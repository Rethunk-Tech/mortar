package devsvc

import (
	"slices"
	"testing"
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
