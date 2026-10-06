package nativehost

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// FuzzServe holds that no byte stream from a browser makes the message loop panic, whatever the frames and their JSON.
func FuzzServe(f *testing.F) {
	testfs.DataHome(f)
	frame := func(body string) []byte {
		n, err := frameLen(len(body))
		if err != nil {
			f.Fatal(err)
		}
		var b bytes.Buffer
		_ = binary.Write(&b, binary.NativeEndian, n)
		b.WriteString(body)
		return b.Bytes()
	}
	f.Add(frame(`{"link":"nxm://stardewvalley/mods/1/files/2"}`))
	f.Add(frame(`{"type":"mod","source":"nexus","sourceGameKey":"stardewvalley","modId":1,"protocol":1}`))
	f.Add(frame(`{"type":"installPackage","source":"thunderstore","sourceGameKey":"lethal-company","package":"a-b"}`))
	f.Add(frame(`{"type":"installed","protocol":-1}`))
	f.Add([]byte{0xff, 0xff, 0xff, 0x7f})
	stub := handlers{
		open:      func(string) error { return nil },
		installed: func(string) []int { return []int{1} },
		im:        func(string, int) (modInProfile, []modInProfile) { return modInProfile{}, nil },
		state:     func(string) (string, string) { return stateReady, "P" },
		updates:   func(string) (string, []modUpdate) { return "", nil },
		broken:    func(string) []int { return nil },
		problems:  func(string, int) []modProblem { return nil },
		requirements: func(string, int) []requirementItem {
			return nil
		},
		packages:       func(string) (string, string, []string) { return stateReady, "P", nil },
		installPackage: func(string, string) error { return nil },
	}
	f.Fuzz(func(t *testing.T, in []byte) {
		_ = serveHandlers(bytes.NewReader(in), io.Discard, stub)
	})
}
