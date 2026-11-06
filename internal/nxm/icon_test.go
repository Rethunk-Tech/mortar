//go:build !windows

package nxm

import (
	"bytes"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

// The embedded icons are copies of the build's, which go:embed cannot reach from this package.
func TestEmbeddedIconsMatchTheBuildIcons(t *testing.T) {
	for path, embedded := range map[string][]byte{"../../build/appicon.svg": iconSVG, "../../build/linux/icons/tech.rethunk.Mortar-256.png": iconPNG} {
		want, err := safefs.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(embedded, want) {
			t.Errorf("internal/nxm/icons differs from %s: copy it over", path)
		}
	}
}
