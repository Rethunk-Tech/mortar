package smapi

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testmain"
)

// Scratch folders are made under the data folder's cache, which a test must not point at the real one.
func TestMain(m *testing.M) {
	testmain.RunInTempData(m, "smapi")
}
