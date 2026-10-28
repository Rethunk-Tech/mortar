package configsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFieldsAndSetFieldReachABepInExFile(t *testing.T) {
	s, path := newService(t)
	const id = "com.example.betterstuff"
	fields, err := s.Fields("lc", "p", id)
	if err != nil || len(fields) == 0 {
		t.Fatalf("fields %+v %v", fields, err)
	}
	var delay string
	for _, f := range fields {
		if f.Path == "com.example.betterstuff.cfg/General/LeaveDelay" {
			delay = f.Value
		}
	}
	if delay != "120" {
		t.Fatalf("LeaveDelay = %q in %+v", delay, fields)
	}
	if ok, err := s.SetField("lc", "p", id, "com.example.betterstuff.cfg/General/LeaveDelay", "60"); !ok || err != nil {
		t.Fatalf("set: %v %v", ok, err)
	}
	if raw, _ := os.ReadFile(filepath.Clean(path)); !strings.Contains(string(raw), "LeaveDelay = 60") {
		t.Fatalf("not written:\n%s", raw)
	}
	if ok, err := s.SetField("lc", "p", id, "com.example.betterstuff.cfg/General/LeaveDelay", "9999"); !ok || err == nil {
		t.Fatalf("out of range must be refused: %v %v", ok, err)
	}
	if ok, _ := s.SetField("lc", "p", id, "nested.foo", "2"); ok {
		t.Fatal("a content pack path is not a typed file")
	}
}

func TestPathEscapesDotsInKeys(t *testing.T) {
	t.Parallel()
	if got := path("Keys", "Open.Menu"); got != `Keys.Open\.Menu` {
		t.Fatalf("path = %q", got)
	}
}
