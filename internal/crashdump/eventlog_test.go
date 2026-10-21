package crashdump

import (
	"os"
	"strings"
	"testing"
)

const eventsXML = `<Events>
<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event"><System><Provider Name=".NET Runtime"/><EventID Qualifiers="0">1026</EventID><TimeCreated SystemTime="2026-10-21T17:30:43.1Z"/></System><EventData><Data>Application: Cat Mail Co.exe
CoreCLR Version: 8.0.1024.46610
.NET Version: 8.0.10
Description: The process was terminated due to an unhandled exception.
Exception Info: System.NullReferenceException: Object reference not set to an instance of an object.
   at System.Linq.Enumerable.First(IEnumerable source)
   at BepInEx.Logging.Logger.Log()
   at BetterRepair.Logic.CardboardStock.Refill(Int32 count)
   at UnityEngine.Events.UnityEvent.Invoke()
</Data></EventData></Event>
<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event"><System><Provider Name="Application Error"/><EventID Qualifiers="0">1000</EventID></System><EventData><Data>Cat Mail Co.exe</Data><Data>2022.3.1.0</Data><Data>67a1b2c3</Data><Data>KERNELBASE.dll</Data><Data>10.0.22621.4249</Data><Data>5f6e7d8c</Data><Data>c0000005</Data><Data>00000000001c408a</Data><Data>4a38</Data></EventData></Event>
<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event"><System><Provider Name="Application Error"/><EventID Qualifiers="0">1000</EventID></System><EventData><Data>explorer.exe</Data><Data>10.0</Data><Data>1</Data><Data>ntdll.dll</Data><Data>10.0</Data><Data>1</Data><Data>c0000005</Data><Data>00000000000a0b0c</Data><Data>1</Data></EventData></Event>
</Events>`

func TestParseEventLog(t *testing.T) {
	got := ParseEventLog([]byte(eventsXML), []string{"cat mail co.exe"})
	if len(got) != 2 {
		t.Fatalf("got %d faults, want 2 (the explorer.exe event is another program's): %+v", len(got), got)
	}
	managed := got[0]
	if managed.Detail != "System.NullReferenceException: Object reference not set to an instance of an object." ||
		managed.Frame != "at BetterRepair.Logic.CardboardStock.Refill(Int32 count)" || managed.Module != "" || managed.Managed == "" {
		t.Errorf("managed fault = %+v", managed)
	}
	native := got[1]
	if native.Module != "KERNELBASE.dll" || native.Detail != "exception c0000005 at offset 00000000001c408a" || !native.HasOffset || native.Offset != 0x1c408a {
		t.Errorf("native fault = %+v", native)
	}
}

func TestParseEventLogKeepsEventsBeforeATruncation(t *testing.T) {
	cut := eventsXML[:len(eventsXML)-400]
	if got := ParseEventLog([]byte(cut), []string{"Cat Mail Co.exe"}); len(got) != 1 {
		t.Errorf("got %d faults from a truncated stream, want the 1 complete one", len(got))
	}
	if got := ParseEventLog([]byte("not xml \x00<<"), []string{"a.exe"}); got != nil {
		t.Errorf("garbage gave %+v", got)
	}
}

// testdata/wevtutil-win11.xml is wevtutil's answer to EventLog's own query on Windows 11 26H2 after a .NET
// Framework program died of a NullReferenceException: real events quote attributes with ' and carry both providers.
func TestEventLogReadsRealWevtutilOutput(t *testing.T) {
	raw, err := os.ReadFile("testdata/wevtutil-win11.xml")
	if err != nil {
		t.Fatal(err)
	}
	faults := ParseEventLog(raw, []string{"CrashTest.exe"})
	if len(faults) != 2 {
		t.Fatalf("faults %+v", faults)
	}
	var managed bool
	for _, f := range faults {
		if strings.Contains(f.Detail, "System.NullReferenceException") {
			managed = true
		}
	}
	if !managed {
		t.Errorf("no managed exception in %+v", faults)
	}
	if got := ParseEventLog(raw, []string{"Other.exe"}); len(got) != 0 {
		t.Errorf("another exe's events were kept: %+v", got)
	}
}
