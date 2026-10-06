package savessvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// A game that is enabled but not installed has no saves folder to resolve; Mortar must still start.
func TestNewServiceStartsWithAGameNotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(home, nil, store, nil); err != nil {
		t.Fatalf("NewService = %v", err)
	}
}
