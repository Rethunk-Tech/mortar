package steam

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andygrunwald/vdf"
)

const localConfig = `"UserLocalConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"413150"
					{
						"LastPlayed"		"1"
						"LaunchOptions"		"-old"
					}
					"440"
					{
						"cloud"
						{
							"last_sync_state"		"synchronized"
						}
					}
				}
			}
		}
	}
	"friends"
	{
		"x"		"y"
	}
}
`

func launchOpts(t *testing.T, b []byte, app string) string {
	t.Helper()
	m, err := vdf.NewParser(bytes.NewReader(b)).Parse()
	if err != nil {
		t.Fatalf("result does not parse: %v\n%s", err, b)
	}
	node := m
	for _, k := range []string{"UserLocalConfigStore", "Software", "Valve", "Steam", "apps", app} {
		next, ok := node[k].(map[string]any)
		if !ok {
			return "<missing>"
		}
		node = next
	}
	v, _ := node["LaunchOptions"].(string)
	return v
}

func TestSetLaunchOptionsEditsOnlyTheOneValue(t *testing.T) {
	line := `"C:\Games\Stardew Valley\StardewModdingAPI.exe" %command%`
	merge := func(old string) string { return strings.TrimSpace(line + " " + old) }

	out, value, err := setLaunchOptions([]byte(localConfig), "413150", merge)
	if err != nil {
		t.Fatal(err)
	}
	if got := launchOpts(t, out, "413150"); got != line+" -old" || value != got {
		t.Fatalf("replaced value = %q (returned %q)", got, value)
	}
	if !bytes.Contains(out, []byte(`"friends"`)) || !bytes.Contains(out, []byte(`"last_sync_state"		"synchronized"`)) {
		t.Fatal("other content must be kept")
	}

	out, _, err = setLaunchOptions([]byte(localConfig), "440", merge)
	if err != nil {
		t.Fatal(err)
	}
	if got := launchOpts(t, out, "440"); got != line {
		t.Fatalf("added key = %q", got)
	}

	out, _, err = setLaunchOptions([]byte(localConfig), "999", merge)
	if err != nil {
		t.Fatal(err)
	}
	if got := launchOpts(t, out, "999"); got != line {
		t.Fatalf("new app block = %q", got)
	}
	if got := launchOpts(t, out, "413150"); got != "-old" {
		t.Fatalf("another app changed: %q", got)
	}

	if _, _, err := setLaunchOptions([]byte(`"UserLocalConfigStore" { }`), "413150", merge); err == nil {
		t.Fatal("a file without an apps section must be refused")
	}
}
