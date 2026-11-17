package installer

import (
	"testing"
)

func TestMinecraftArchivesLayOutAsJars(t *testing.T) {
	t.Parallel()
	g := Game{Loaders: []string{"fabric"}, Targets: []Target{{ID: TargetMods}}}
	l, err := (zipped{}).Layout(Archive{Key: "sodium"}, g, Choices{})
	if err != nil || len(l.Files) != 1 || l.Files[0].Rel != "sodium.jar" || !l.Files[0].Zip {
		t.Fatalf("layout %+v %v", l, err)
	}
	if i, ok := Pick(Archive{Key: "sodium"}, g); !ok || i.ID() != DriverZipped {
		t.Fatalf("pick %v %v", i, ok)
	}
}
