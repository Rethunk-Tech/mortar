package devsvc

import "testing"

func TestBatchCommandLine(t *testing.T) {
	got, err := batchCommandLine(`C:\W\cmd.exe`, `C:\Users\Ann Lee\.dotnet\tools\ilspycmd.cmd`, []string{"-t", "Foo+Bar", `C:\a & b\G.dll`})
	want := `"C:\W\cmd.exe" /d /s /c ""C:\Users\Ann Lee\.dotnet\tools\ilspycmd.cmd" "-t" "Foo+Bar" "C:\a & b\G.dll""`
	if err != nil || got != want {
		t.Fatalf("got %s, %v\nwant %s", got, err, want)
	}
	for _, bad := range []string{`a"b`, "100%", "a\nb"} {
		if _, err := batchCommandLine("cmd.exe", `C:\t.cmd`, []string{bad}); err == nil {
			t.Errorf("%q reached cmd.exe", bad)
		}
	}
	if _, err := batchCommandLine("cmd.exe", `C:\50%\t.cmd`, nil); err == nil {
		t.Error("a tool path with % reached cmd.exe")
	}
}
