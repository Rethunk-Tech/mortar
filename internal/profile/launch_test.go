package profile

import (
	"reflect"
	"testing"
)

func TestLaunchSettingsParsing(t *testing.T) {
	args, err := LaunchPrefixArgs(`gamemoderun "mango hud" 'with space'`)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"gamemoderun", "mango hud", "with space"}; !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
	if _, err := LaunchPrefixArgs(`unterminated"`); err == nil {
		t.Fatal("unterminated quote accepted")
	}
	if _, err := LaunchEnvironment("GOOD=value\n_BAD=value"); err == nil {
		t.Fatal("invalid environment accepted")
	}
	env, err := LaunchEnvironment("GOOD=value\nALSO=two=parts")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"GOOD=value", "ALSO=two=parts"}; !reflect.DeepEqual(env, want) {
		t.Fatalf("env = %#v, want %#v", env, want)
	}
}
