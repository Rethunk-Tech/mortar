package profile

import "testing"

// A history snapshot names an entry the update replaced, so the keys it held before the update still identify it.
func TestEntryUsesKeyMatchesPreviousKeys(t *testing.T) {
	t.Parallel()
	e := Entry{
		Key:                    "current",
		PreviousKey:            "before",
		ExtraStoreKeys:         []string{"extra-current"},
		PreviousExtraStoreKeys: []string{"extra-before"},
	}
	for _, key := range []string{"before", "extra-before"} {
		if !entryUsesKey(e, map[string]struct{}{key: {}}) {
			t.Fatalf("entryUsesKey missed the previous key %q", key)
		}
	}
	if entryUsesKey(e, map[string]struct{}{"someone-else": {}}) {
		t.Fatal("entryUsesKey matched a key the entry never held")
	}
}
