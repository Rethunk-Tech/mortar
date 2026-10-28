package profile

import (
	"strings"
	"testing"
)

func TestSetJSONValueKeepsADottedKeyWhole(t *testing.T) {
	t.Parallel()
	out, err := SetJSONValue([]byte(`{"Keys":{"Open.Menu":"K"}}`), `Keys.Open\.Menu`, "F1")
	if err != nil || !strings.Contains(string(out), `"Open.Menu": "F1"`) || strings.Contains(string(out), `"Open": {`) {
		t.Fatalf("%s %v", out, err)
	}
}
