package jsonc

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"unicode/utf16"
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

func utf16Of(s string, order binary.AppendByteOrder, bom []byte) []byte {
	out := slices.Clone(bom)
	for _, u := range utf16.Encode([]rune(s)) {
		out = order.AppendUint16(out, u)
	}
	return out
}

func TestCleanNewtonsoftForms(t *testing.T) {
	const want = `{"a":"x y","b":[1,null]}`
	for name, in := range map[string][]byte{
		"strict":             []byte(`{"a":"x y","b":[1,null]}`),
		"single quotes":      []byte(`{'a':'x y','b':[1,null]}`),
		"unquoted names":     []byte(`{a:"x y", b : [1,null]}`),
		"mixed":              []byte(`{ a: 'x y', 'b': [1, null,], // note` + "\n}"),
		"utf8 bom":           []byte("\xef\xbb\xbf" + `{"a":"x y","b":[1,null]}`),
		"utf16 le bom":       utf16Of(`{"a":"x y","b":[1,null]}`, binary.LittleEndian, []byte{0xff, 0xfe}),
		"utf16 be bom":       utf16Of(`{"a":"x y","b":[1,null]}`, binary.BigEndian, []byte{0xfe, 0xff}),
		"NaN and undefined":  []byte(`{a:"x y",b:[1,undefined]}`),
		"infinity":           []byte(`{a:"x y",b:[1,-Infinity]}`),
		"block comment keys": []byte(`{/* c */a:"x y",b:[1,NaN]}`),
	} {
		var got, exp any
		if err := json.Unmarshal(Clean(in), &got); err != nil {
			t.Errorf("%s: %v (cleaned %q)", name, err, Clean(in))
			continue
		}
		_ = json.Unmarshal([]byte(want), &exp)
		if !reflect.DeepEqual(got, exp) {
			t.Errorf("%s: got %v", name, got)
		}
	}
}

func TestCleanStringContents(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"literal newline and tab": {"{\"a\":\"l1\nl2\tx\x01\"}", "l1\nl2\tx\x01"},
		"apostrophe escape":       {`{'a':'it\'s "q"'}`, `it's "q"`},
		"comment markers":         {`{"a":"// not /* a */ comment, ]"}`, `// not /* a */ comment, ]`},
		"unquoted lookalike":      {`{"a":"b:c, d: e"}`, `b:c, d: e`},
		"escapes survive":         {`{"a":"\u00e9\n\"\\"}`, "\u00e9\n\"\\"},
		"single in double":        {`{"a":"don't"}`, `don't`},
	} {
		var got map[string]string
		if err := json.Unmarshal(Clean([]byte(tc.in)), &got); err != nil {
			t.Errorf("%s: %v", name, err)
		} else if got["a"] != tc.want {
			t.Errorf("%s: got %q want %q", name, got["a"], tc.want)
		}
	}
}

func TestCleanKeepsBlockCommentNewlines(t *testing.T) {
	if got := bytes.Count(Clean([]byte("{/*\n\n*/ \"a\": 1}")), []byte("\n")); got != 2 {
		t.Fatalf("newlines = %d", got)
	}
}
