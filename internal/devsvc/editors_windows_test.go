//go:build windows

package devsvc

import (
	"os"
	"slices"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestAssociatedEditorsReadsTheProgramsRegisteredForAFileType(t *testing.T) {
	const (
		fileType = ".mortareditortest"
		progID   = "MortarEditorTest.File"
	)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	classes := `Software\Classes\`
	create := func(path string) registry.Key {
		key, _, err := registry.CreateKey(registry.CURRENT_USER, classes+path, registry.SET_VALUE)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = key.Close()
			_ = registry.DeleteKey(registry.CURRENT_USER, classes+path)
		})
		return key
	}
	// Cleanups run last to first, so each key goes before the parent it was created under.
	create(fileType)
	if err := create(fileType+`\OpenWithProgids`).SetStringValue(progID, ""); err != nil {
		t.Fatal(err)
	}
	create(progID)
	create(progID + `\shell`)
	create(progID + `\shell\open`)
	if err := create(progID+`\shell\open\command`).SetStringValue("", `"`+exe+`" "%1"`); err != nil {
		t.Fatal(err)
	}
	found := associatedEditors([]string{fileType})
	if !slices.ContainsFunc(found, func(e Editor) bool { return e.Path == exe }) {
		t.Fatalf("%s not found in %v", exe, found)
	}
}
