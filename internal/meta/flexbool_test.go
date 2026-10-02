package meta

import (
	"encoding/json"
	"testing"
)

func TestRawDependencyReadsStringIsRequired(t *testing.T) {
	for in, want := range map[string]bool{`"false"`: false, `"FALSE"`: false, `false`: false, `"true"`: true, `true`: true} {
		var d rawDependency
		if err := json.Unmarshal([]byte(`{"UniqueID":"X","IsRequired":`+in+`}`), &d); err != nil {
			t.Fatal(err)
		}
		if d.IsRequired == nil || bool(*d.IsRequired) != want {
			t.Errorf("IsRequired %s = %v, want %v", in, d.IsRequired, want)
		}
	}
}
