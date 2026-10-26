package share

import (
	"bytes"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestSharedProfilesLeaveDevHelpersOut(t *testing.T) {
	p := sample()
	p.Entries = append(p.Entries, nexus("dev", 99, 1))
	p.Entries[len(p.Entries)-1].DevOnly = true
	base, _, _ := Collect(sample())
	want := len(base.Entries)

	producers := map[string]func(profile.Profile) ([]Ref, error){
		"link": func(p profile.Profile) ([]Ref, error) {
			s, _, _ := Collect(p)
			return s.Entries, nil
		},
		"file": func(p profile.Profile) ([]Ref, error) {
			var buf bytes.Buffer
			if _, err := Write(&buf, "stardew", p, t.TempDir()); err != nil {
				return nil, err
			}
			pv, err := ReadBytes(buf.Bytes())
			return pv.Entries, err
		},
	}
	for name, produce := range producers {
		t.Run(name, func(t *testing.T) {
			refs, err := produce(p)
			if err != nil {
				t.Fatal(err)
			}
			if len(refs) != want {
				t.Fatalf("got %d refs, want %d without the dev helper", len(refs), want)
			}
			for _, r := range refs {
				if r.ModID == 99 {
					t.Fatalf("dev helper shared: %+v", r)
				}
			}
		})
	}
}
