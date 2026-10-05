package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestOverlayDefaultsAndPortRange(t *testing.T) {
	s, dir := open(t)
	got := s.Get()
	if got.OverlayEnabled || got.OverlayPort != DefaultOverlayPort || got.OverlayToken != "" {
		t.Fatalf("defaults = %+v", got)
	}
	if _, err := s.Update(func(v *Settings) { v.OverlayPort = MinOverlayPort - 1 }); err == nil {
		t.Fatal("port 1023 accepted")
	}
	if _, err := s.Update(func(v *Settings) { v.OverlayPort = MaxOverlayPort + 1 }); err == nil {
		t.Fatal("port 65536 accepted")
	}
	if _, err := s.Update(func(v *Settings) { v.OverlayPort = 9000 }); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"global":{"accent":"sand","background":"image","overlayPort":80}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if s2.Get().OverlayPort != DefaultOverlayPort {
		t.Fatalf("load port = %d", s2.Get().OverlayPort)
	}
}

func TestNewOverlayTokenIs32BytesHex(t *testing.T) {
	a, err := NewOverlayToken()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewOverlayToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 64 || len(b) != 64 {
		t.Fatalf("len %d %d", len(a), len(b))
	}
	if a == b {
		t.Fatal("tokens were equal")
	}
	for _, r := range a {
		ok := r >= '0' && r <= '9' || r >= 'a' && r <= 'f'
		if !ok {
			t.Fatalf("not hex: %q", a)
		}
	}
}

func TestSetOverlayEnabledGeneratesTokenOnce(t *testing.T) {
	s, _ := open(t)
	svc := NewService(s)
	if err := svc.SetOverlayEnabled(true); err != nil {
		t.Fatal(err)
	}
	first := s.Get().OverlayToken
	if first == "" || !s.Get().OverlayEnabled {
		t.Fatalf("%+v", s.Get())
	}
	if err := svc.SetOverlayEnabled(false); err != nil {
		t.Fatal(err)
	}
	if s.Get().OverlayEnabled || s.Get().OverlayToken != first {
		t.Fatalf("disable changed token: %+v", s.Get())
	}
	if err := svc.SetOverlayEnabled(true); err != nil {
		t.Fatal(err)
	}
	if s.Get().OverlayToken != first {
		t.Fatal("re-enable rotated the token")
	}
	if err := svc.RegenerateOverlayToken(); err != nil {
		t.Fatal(err)
	}
	if s.Get().OverlayToken == first || len(s.Get().OverlayToken) != 64 {
		t.Fatalf("regenerate = %q", s.Get().OverlayToken)
	}
}

func TestOverlayTokenStaysOutOfJSONLogsShape(t *testing.T) {
	s, dir := open(t)
	svc := NewService(s)
	if err := svc.SetOverlayEnabled(true); err != nil {
		t.Fatal(err)
	}
	raw, err := fsx.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	global, _ := m["global"].(map[string]any)
	tok, _ := global["overlayToken"].(string)
	if tok == "" {
		t.Fatal("token not stored")
	}
	if strings.Contains(string(raw), "OverlayToken") {
		t.Fatal("PascalCase token key in settings.json")
	}
}
