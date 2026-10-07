package crashdump

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unicode/utf16"
)

type testModule struct {
	name string
	base uint64
	size uint32
}

// u32 narrows a test fixture's offset or count, which always fits.
func u32(n int) uint32 {
	if n < 0 || n > math.MaxUint32 {
		panic("fixture offset out of range")
	}
	return uint32(n)
}

// minidump builds the header, directory, exception stream and module list of a minidump.
func minidump(addr uint64, mods ...testModule) []byte {
	le := binary.LittleEndian
	const dirRva, excRva, modRva = 32, 32 + 2*12, 32 + 2*12 + 168
	var buf bytes.Buffer
	put := func(v any) { _ = binary.Write(&buf, le, v) }
	put(uint32(minidumpMagic))
	put(uint32(0xa793))
	put(uint32(2))
	put(uint32(dirRva))
	put(make([]byte, 16))
	put([]uint32{exceptionStream, 168, excRva})
	put([]uint32{moduleListStream, u32(4 + len(mods)*moduleEntrySize), modRva})
	exc := make([]byte, 168)
	le.PutUint32(exc[8:], 0xc0000005)
	le.PutUint64(exc[24:], addr)
	buf.Write(exc)
	put(u32(len(mods)))
	strAt := modRva + 4 + len(mods)*moduleEntrySize
	var strs bytes.Buffer
	for _, m := range mods {
		entry := make([]byte, moduleEntrySize)
		le.PutUint64(entry[0:], m.base)
		le.PutUint32(entry[8:], m.size)
		le.PutUint32(entry[20:], u32(strAt+strs.Len()))
		buf.Write(entry)
		u := utf16.Encode([]rune(m.name))
		_ = binary.Write(&strs, le, u32(len(u)*2))
		_ = binary.Write(&strs, le, u)
	}
	buf.Write(strs.Bytes())
	return buf.Bytes()
}

func TestMinidumpNamesTheModuleHoldingTheFaultAddress(t *testing.T) {
	t.Parallel()
	dump := minidump(0x7ff800001234,
		testModule{`C:\Game\Game.exe`, 0x140000000, 0x2000000},
		testModule{`C:\Game\Mods\Cool\Cool.dll`, 0x7ff800000000, 0x10000},
		testModule{`C:\Windows\System32\ntdll.dll`, 0x7ff900000000, 0x200000},
	)
	got, err := ParseMinidump(bytes.NewReader(dump), int64(len(dump)))
	if err != nil || got.Module != "Cool.dll" {
		t.Fatalf("fault = %+v, %v", got, err)
	}
	if got.Detail != "exception 0xc0000005 at 0x7ff800001234" {
		t.Fatalf("detail = %q", got.Detail)
	}
}

func TestMinidumpWithAnAddressInNoModuleNamesNone(t *testing.T) {
	t.Parallel()
	dump := minidump(0x1, testModule{`C:\Game\Game.exe`, 0x140000000, 0x2000000})
	got, err := ParseMinidump(bytes.NewReader(dump), int64(len(dump)))
	if err != nil || got.Module != "" {
		t.Fatalf("fault = %+v, %v", got, err)
	}
}

func TestMinidumpRejectsGarbageAndTruncation(t *testing.T) {
	t.Parallel()
	for name, data := range map[string][]byte{
		"empty":     nil,
		"not a dmp": bytes.Repeat([]byte("x"), 200),
		"truncated": minidump(0x1, testModule{"a.dll", 0, 10})[:60],
	} {
		if _, err := ParseMinidump(bytes.NewReader(data), int64(len(data))); err == nil {
			t.Errorf("%s: parsed", name)
		}
	}
}

const unityLog = `Unity Player [version: Unity 2022.3.9f1]

========== OUTPUTTING STACK TRACE ==================

0x00007FFB2B3A1234 (KERNELBASE) RaiseException
0x00007FFA11112222 (CoolMod) CoolMod::Native
0x00007FF6A1B2C3D4 (UnityPlayer) UnityMain
0x00007FFC00001111 (KERNEL32) BaseThreadInitThunk

========== END OF STACKTRACE ===========
`

func TestUnityErrorLogNamesTheFirstNonSystemFrame(t *testing.T) {
	t.Parallel()
	got, ok := ParseUnityErrorLog(unityLog)
	if !ok || got.Module != "CoolMod" {
		t.Fatalf("fault = %+v, %v", got, ok)
	}
	if _, ok := ParseUnityErrorLog("nothing native here"); ok {
		t.Fatal("a log without frames named a module")
	}
}

const coredumpInfo = `           PID: 4242 (StardewModdingA)
        Signal: 11 (SEGV)
Stack trace of thread 4242:
#0  0x00007f0000001000 raise (libc.so.6 + 0x1000)
#1  0x00007f0000002000 n/a (libCoolNative.so + 0x2000)
#2  0x00007f0000003000 mono_runtime_invoke (libmonosgen-2.0.so.1 + 0x3000)
`

func TestCoredumpInfoNamesTheFirstNonSystemModule(t *testing.T) {
	t.Parallel()
	got, ok := ParseCoredumpInfo(coredumpInfo)
	if !ok || got.Module != "libCoolNative.so" {
		t.Fatalf("fault = %+v, %v", got, ok)
	}
}

func TestWindowsReadsOnlyDumpsNewerThanTheRun(t *testing.T) {
	t.Parallel()
	local := t.TempDir()
	crashDumps := filepath.Join(local, "CrashDumps")
	crash := filepath.Join(local, "Temp", "Co", "Prod", "Crashes", "Crash_1")
	for _, d := range []string{crashDumps, crash} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	dump := minidump(0x10, testModule{`C:\Mods\Old.dll`, 0, 0x100})
	oldDump := filepath.Join(crashDumps, "Game.exe.100.dmp")
	newDump := filepath.Join(crashDumps, "Game.exe.200.dmp")
	for _, p := range []string{oldDump, newDump} {
		if err := os.WriteFile(p, dump, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(crash, "error.log"), []byte(unityLog), 0o600); err != nil {
		t.Fatal(err)
	}
	since := time.Now().Add(-time.Hour)
	if err := os.Chtimes(oldDump, since.Add(-time.Hour), since.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	got := Windows(crashDumps, filepath.Join(local, "Temp"), []string{"Game.exe"}, since)
	if len(got) != 2 || got[0].Module != "CoolMod" || got[1].Module != "Old.dll" {
		t.Fatalf("faults = %+v, want the Unity folder's then the one fresh WER dump", got)
	}
	if got := Windows(crashDumps, filepath.Join(local, "Temp"), []string{"Other.exe"}, since); len(got) != 1 {
		t.Fatalf("another exe's dumps were read: %+v", got)
	}
}
