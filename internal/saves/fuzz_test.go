package saves

import (
	"bytes"
	"maps"
	"testing"
	"testing/iotest"
)

// FuzzSaveFile holds that no save file makes the key scan or the SaveGameInfo read panic, and that the key scan finds
// the same keys and farm whether the file arrives at once or a byte at a time.
func FuzzSaveFile(f *testing.F) {
	f.Add([]byte(`<SaveGame><player><name>Ann</name><farmName>Hill</farmName><money>5</money></player>` +
		`<whichFarm>MeadowlandsFarm</whichFarm><item><key><string>Me.Mod_Item</string></key></item></SaveGame>`))
	f.Add([]byte("\xef\xbb\xbf<Farmer><name>A</name><dayOfMonthForSaveGame>3</dayOfMonthForSaveGame></Farmer>"))
	f.Add([]byte(`<key><string>`))
	f.Add([]byte(`<whichFarm>7</whichFarm><key><string>a</string></key><key><string>a</string></key>`))
	f.Fuzz(func(t *testing.T, b []byte) {
		var info Info
		readInfo(b, &info)
		whole, farm, err := distinctKeys(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		bytewise, farmBytewise, err := distinctKeys(iotest.OneByteReader(bytes.NewReader(b)))
		if err != nil {
			t.Fatal(err)
		}
		if !maps.Equal(whole, bytewise) || farm != farmBytewise {
			t.Fatalf("whole %v %+v, a byte at a time %v %+v", whole, farm, bytewise, farmBytewise)
		}
	})
}
