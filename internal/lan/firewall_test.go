package lan

import (
	"encoding/base64"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestScriptWithExeQuotesAndEncodes(t *testing.T) {
	t.Parallel()
	exe := `C:\Users\Zoë O'Brien\AppData\Local\Programs\Mortar\mortar.exe`
	raw, err := base64.StdEncoding.DecodeString(encodeCommand(scriptWithExe("body", exe)))
	if err != nil {
		t.Fatal(err)
	}
	units := make([]uint16, len(raw)/2)
	for i := range units {
		units[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
	}
	want := "$exe = 'C:\\Users\\Zoë O''Brien\\AppData\\Local\\Programs\\Mortar\\mortar.exe'\nbody"
	if got := string(utf16.Decode(units)); got != want {
		t.Fatalf("decoded script = %q, want %q", got, want)
	}
	if strings.ContainsAny(encodeCommand("x"), "'\n") {
		t.Fatal("encoded payload is not plain base64")
	}
}
