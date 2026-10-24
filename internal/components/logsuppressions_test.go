package components

import "testing"

func TestDecodeLogSuppressionsRefusesAnotherGameAndAnUnlabelledEntry(t *testing.T) {
	for name, body := range map[string]string{
		"another game": `{"game":"valheim","serial":1,"entries":[]}`,
		"no serial":    `{"game":"lc","entries":[]}`,
		"no label":     `{"game":"lc","serial":1,"entries":[{"source":"T::Open"}]}`,
	} {
		if _, err := DecodeLogSuppressions([]byte(body), "lc"); err == nil {
			t.Errorf("%s: decoded", name)
		}
	}
	l, err := DecodeLogSuppressions([]byte(`{"game":"lc","serial":2,"entries":[{"source":"T::Open","label":"Door"}]}`), "lc")
	if err != nil || len(l.Entries) != 1 {
		t.Fatalf("list = %+v, %v", l, err)
	}
}
