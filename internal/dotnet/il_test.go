package dotnet

import (
	"slices"
	"testing"
)

func TestOpsDecodesOperandsAndTwoByteOpcodes(t *testing.T) {
	// ldc.i4.s 5; ldloc 0x0102 (FE 0C); call 0x0A000001; ret
	code := []byte{0x1F, 0x05, 0xFE, 0x0C, 0x02, 0x01, 0x28, 0x01, 0x00, 0x00, 0x0A, 0x2A}
	var got []instr
	for in := range ops(code) {
		got = append(got, in)
	}
	want := []instr{
		{start: 0, op: 0x1F, at: 1, next: 2, arg: 5},
		{start: 2, op: 0xFE0C, at: 4, next: 6, arg: 0x0102},
		{start: 6, op: 0x28, at: 7, next: 11, arg: 0x0A000001},
		{start: 11, op: 0x2A, at: 12, next: 12},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("ops = %+v", got)
	}
	if b := ilCode(append([]byte{12<<2 | 2}, code...)); !slices.Equal(b, code) {
		t.Fatalf("ilCode tiny = % x", b)
	}
}

func TestReturnTypeSkipsGenericAndParamCounts(t *testing.T) {
	if got := returnType([]byte("\x00\x00\x01")); got != elementVoid {
		t.Fatalf("void = %#x", got)
	}
	if got := returnType([]byte("\x10\x01\x02\x02")); got != elementBool {
		t.Fatalf("generic bool = %#x", got)
	}
	if got := returnType([]byte("\x00")); got != 0 {
		t.Fatalf("short = %#x", got)
	}
}
