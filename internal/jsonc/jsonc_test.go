package jsonc

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
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

func utf32Of(s string, order binary.AppendByteOrder, bom []byte) []byte {
	out := slices.Clone(bom)
	for _, c := range []byte(s) { // ASCII only
		out = order.AppendUint32(out, uint32(c))
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
		"utf32 le bom":       utf32Of(`{"a":"x y","b":[1,null]}`, binary.LittleEndian, []byte{0xff, 0xfe, 0, 0}),
		"utf32 be bom":       utf32Of(`{"a":"x y","b":[1,null]}`, binary.BigEndian, []byte{0, 0, 0xfe, 0xff}),
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

func TestCleanUnclosedBlockCommentIsInvalid(t *testing.T) {
	if json.Valid(Clean([]byte(`{"o":1} /* x`))) {
		t.Fatal("unclosed block comment accepted")
	}
}

func TestCleanElisions(t *testing.T) {
	for in, want := range map[string]string{
		`[,]`: `[null]`, `[1,,2]`: `[1,null,2]`, `[,1]`: `[null,1]`, `[1,,]`: `[1,null]`, `[1,]`: `[1]`,
		`{"o":,"s":"x"}`: `{"o":null,"s":"x"}`, `{"o":,}`: `{"o":null}`,
	} {
		if got := Clean([]byte(in)); string(got) != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
	for _, in := range []string{`{,}`, `{"o":1,,"s":"x"}`, `{"o":1,,}`} {
		if json.Valid(Clean([]byte(in))) {
			t.Errorf("%s accepted", in)
		}
	}
}

func TestCleanKeepsLinesAcrossMultilineStrings(t *testing.T) {
	var se *json.SyntaxError
	err := json.Unmarshal(Clean([]byte("{\"a\":\"x\ny\nz\",\n\"b\": }")), new(any))
	if !errors.As(err, &se) {
		t.Fatal(err)
	}
	line := 1 + bytes.Count(Clean([]byte("{\"a\":\"x\ny\nz\",\n\"b\": }"))[:se.Offset], []byte("\n"))
	if line != 4 {
		t.Fatalf("error on line %d", line)
	}
}

func TestCleanBareDecimalPoints(t *testing.T) {
	for in, want := range map[string]string{`[.5,-.5,5.,1.5,5.e2]`: `[0.5,-0.5,5.0,1.5,5.0e2]`} {
		if got := Clean([]byte(in)); string(got) != want {
			t.Errorf("got %q want %q", got, want)
		}
	}
	if json.Valid(Clean([]byte(`[+1]`))) {
		t.Error("+1 accepted")
	}
}
