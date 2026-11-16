package profile

import "testing"

func TestSetJSONValueKeepsADottedKeyWhole(t *testing.T) {
	t.Parallel()
	out, err := SetJSONValue([]byte(`{"Keys":{"Open.Menu":"K"}}`), `Keys.Open\.Menu`, "F1")
	if err != nil || string(out) != `{"Keys":{"Open.Menu":"F1"}}` {
		t.Fatalf("%s %v", out, err)
	}
}

func TestSetJSONValueKeepsTheUsersOtherBytes(t *testing.T) {
	t.Parallel()
	in := "{\n\t// my note\n\t\"A\":   1,\n\t\"B\": \"x\"\n}\n"
	out, err := SetJSONValue([]byte(in), "B", "y")
	want := "{\n\t// my note\n\t\"A\":   1,\n\t\"B\": \"y\"\n}\n"
	if err != nil || string(out) != want {
		t.Fatalf("%q %v", out, err)
	}
}
