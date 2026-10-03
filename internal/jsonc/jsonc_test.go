package jsonc

import (
	"encoding/json"
	"testing"
)

func TestCleanBOMCommentsTrailingComma(t *testing.T) {
	in := []byte("\xef\xbb\xbf{\n  // line\n  \"a\": 1, /* block */\n  \"b\": [2,],\n}\n")
	var got map[string]any
	if err := json.Unmarshal(Clean(in), &got); err != nil {
		t.Fatal(err)
	}
	if got["a"] != float64(1) {
		t.Fatalf("a = %v", got["a"])
	}
	b, ok := got["b"].([]any)
	if !ok || len(b) != 1 || b[0] != float64(2) {
		t.Fatalf("b = %v", got["b"])
	}
}
