package modconfig

import "testing"

// FuzzParse feeds a Content Patcher content.json's ConfigSchema, which the config editor offers as choices and checks
// every edit against.
func FuzzParse(f *testing.F) {
	f.Add([]byte(`{"ConfigSchema":{"Mode":{"AllowValues":"Easy, Hard","Default":"Easy","Section":"S"},"Tags":{"AllowValues":"a,b,,c","AllowMultiple":true,"AllowBlank":true,"Default":true}}}`), "Mode", "hard")
	f.Add([]byte(`{"ConfigSchema":{"N":{"Default":5}}}`), "x.N", " ")
	f.Add([]byte(`{"ConfigSchema":[]}`), "", "")
	f.Fuzz(func(t *testing.T, content []byte, path, value string) {
		s, err := Parse(content)
		if err != nil {
			return
		}
		for name, field := range s {
			if got, ok := s.Lookup(name); !ok || got.Name != field.Name {
				t.Fatalf("Lookup(%q) = %+v, %v", name, got, ok)
			}
		}
		_ = s.Validate(path, value)
	})
}
