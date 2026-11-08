package components

import "testing"

func TestAssetNameFillsVersionAndTag(t *testing.T) {
	cases := map[string]struct {
		pattern, tag, want string
	}{
		"version strips a leading v": {pattern: "SMAPI-{version}-installer.zip", tag: "v4.5.2", want: "SMAPI-4.5.2-installer.zip"},
		"version keeps its own v":    {pattern: "SMAPI-{version}-installer.zip", tag: "4.5.2", want: "SMAPI-4.5.2-installer.zip"},
		"tag keeps the release form": {pattern: "tool-{tag}.zip", tag: "v1.0", want: "tool-v1.0.zip"},
		"both placeholders":          {pattern: "{version}/{tag}.zip", tag: "v1.0", want: "1.0/v1.0.zip"},
		"no placeholder":             {pattern: "MelonLoader.x64.zip", tag: "v0.6.6", want: "MelonLoader.x64.zip"},
	}
	for name, c := range cases {
		if got := (SourceComponent{AssetPattern: c.pattern}).AssetName(c.tag); got != c.want {
			t.Errorf("%s: AssetName = %q, want %q", name, got, c.want)
		}
	}
}
