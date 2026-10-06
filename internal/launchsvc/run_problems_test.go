package launchsvc

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func TestRunProblemsReadTheBepInExRunAndItsOwnPlayerLog(t *testing.T) {
	datadirtest.Use(t, filepath.Join(t.TempDir(), "data"))
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "A")
	dir, err := profiles.ProfileDir("lethal-company", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "BepInEx"), 0o700); err != nil {
		t.Fatal(err)
	}
	const logOutput = "[Info   :   BepInEx] BepInEx 5.4.22 - Lethal Company\n" +
		"[Error  :   BepInEx] Could not load [Foo 1.0.0] because it has missing dependencies: com.bar.lib\n" +
		"[Message:   BepInEx] Chainloader startup complete\n"
	if err := os.WriteFile(filepath.Join(dir, "BepInEx", "LogOutput.log"), []byte(logOutput), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), nil, profiles)
	svc.record(game.Find("lethal-company"), p.ID, time.Now(), false)
	runs, err := svc.Runs("lethal-company", p.ID)
	if err != nil || len(runs) != 1 {
		t.Fatalf("runs = %v, %v", runs, err)
	}
	path, err := svc.runFile("lethal-company", p.ID, runs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	const player = "NullReferenceException: Object reference not set to an instance of an object\n" +
		"  at Baz.Patches.Menu.Start () [0x00000] in <abc>:0\n"
	if err := os.WriteFile(runPlayerLogPath(filepath.Dir(path), runs[0].ID), []byte(player), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := svc.RunProblems("lethal-company", p.ID, runs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	kinds := make([]launch.SMAPIProblemKind, len(got))
	for i, x := range got {
		kinds[i] = x.Kind
	}
	if !slices.Contains(kinds, bepinex5.KindMissingDependency) || !slices.Contains(kinds, bepinex5.KindUnityException) {
		t.Fatalf("problems = %+v", got)
	}
	for _, x := range got {
		if x.Kind == bepinex5.KindMissingDependency && (x.Fix != launch.SMAPIFixInstallDependency || x.Dependency != "com.bar.lib" || x.ModName != "Foo 1.0.0") {
			t.Fatalf("missing dependency row = %+v", x)
		}
	}
}
