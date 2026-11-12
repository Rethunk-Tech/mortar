package devsvc

import (
	"os"
	"testing"
)

// A git hook exports GIT_DIR and friends; left set, every git call the tests make acts on the hook's repo, not the temp one.
func TestMain(m *testing.M) {
	for _, k := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_PREFIX"} {
		_ = os.Unsetenv(k)
	}
	os.Exit(m.Run())
}
