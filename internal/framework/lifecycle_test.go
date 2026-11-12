package framework

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

type statefulFramework struct {
	fakeFramework
	forgot *int
}

func (s statefulFramework) Forget()      { *s.forgot++ }
func (statefulFramework) IndexesAssets() {}

func TestForgetAndFormatQueries(t *testing.T) {
	var forgot int
	Register(statefulFramework{fakeFramework{id: mod.NewID("lifecyclefmt", "X")}, &forgot})
	Forget()
	if forgot != 1 {
		t.Errorf("Forget reached the stateful framework %d times, want 1", forgot)
	}
	for _, c := range []struct {
		formats []string
		want    bool
	}{{[]string{"lifecyclefmt"}, true}, {[]string{"other"}, false}, {nil, false}} {
		if got := HasFrameworks(c.formats); got != c.want {
			t.Errorf("HasFrameworks(%v) = %v", c.formats, got)
		}
		if got := IndexesAssets(c.formats); got != c.want {
			t.Errorf("IndexesAssets(%v) = %v", c.formats, got)
		}
	}
}
