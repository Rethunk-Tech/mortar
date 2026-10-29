package winname_test

import (
	"testing"

	ga "github.com/Rethunk-Tech/go-archive"

	"github.com/Rethunk-Tech/mortar/internal/winname"
)

// Mortar hands winname.Valid to the archive module as its segment policy; the module's own WindowsSegment must
// not be laxer than the policy it is replaced by.
func TestValidAgreesWithTheModule(t *testing.T) {
	for _, name := range []string{
		"foo", "a.b", ".hidden", "com10", "com10.txt", "COM", "Stardew Valley (Game)",
		"CON", "CON.txt", "con.", "nul.txt", "PRN", "aux ", "CONIN$", "CONOUT$",
		"COM0", "LPT0", "com1", "LPT9.log", "COM¹", "COM²", "COM³", "LPT¹", "lpt²", "LPT³.txt",
		"foo.", "foo ", "", "a<b", "a>b", `a:b`, `a"b`, "a/b", `a\b`, "a|b", "a?b", "a*b", "a\x01b",
	} {
		if got, want := winname.Valid(name), ga.WindowsSegment(name); got != want {
			t.Errorf("Valid(%q) = %v, module WindowsSegment = %v", name, got, want)
		}
	}
}
