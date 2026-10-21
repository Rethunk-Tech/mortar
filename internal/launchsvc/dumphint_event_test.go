package launchsvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/crashdump"
	"github.com/Rethunk-Tech/mortar/internal/loader/dotnetstack"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestEventLogExceptionBlamesThePluginsMod(t *testing.T) {
	faults := crashdump.ParseEventLog([]byte(`<Events><Event><System><Provider Name=".NET Runtime"/><EventID>1026</EventID></System><EventData><Data>Application: Cat Mail Co.exe
Exception Info: System.NullReferenceException: boom
   at System.Linq.Enumerable.First(IEnumerable source)
   at BetterRepair.Logic.CardboardStock.Refill(Int32 count)
</Data></EventData></Event></Events>`), []string{"Cat Mail Co.exe"})
	if len(faults) != 1 {
		t.Fatalf("faults = %+v", faults)
	}
	mods := []profile.Installed{{Enabled: true, EntryDll: "BetterRepair.dll"}, {Enabled: true, EntryDll: "BoatTweaks.dll"}}
	m, b, ok := blamedMod(dotnetstack.Blames(faults[0].Managed, dotnetstack.Unity), mods)
	if !ok || m.EntryDll != "BetterRepair.dll" || b.Frame != "at BetterRepair.Logic.CardboardStock.Refill(Int32 count)" {
		t.Errorf("blame = %+v %+v %v", m, b, ok)
	}
}
