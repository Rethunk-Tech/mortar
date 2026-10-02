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
	for _, bad := range []string{"GOOD=value\n1BAD=value", "NO_EQUALS", "SP ACE=x"} {
		if _, err := LaunchEnvironment(bad); err == nil {
			t.Fatalf("invalid environment %q accepted", bad)
		}
	}
	env, err := LaunchEnvironment("GOOD=value\n_ALSO=two=parts")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"GOOD=value", "_ALSO=two=parts"}; !reflect.DeepEqual(env, want) {
		t.Fatalf("env = %#v, want %#v", env, want)
	}
}
