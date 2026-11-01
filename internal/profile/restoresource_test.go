package profile

import "testing"

func TestRestoreSourceRejectsMalformedTokens(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	for _, token := range []string{"abc/x", "1/ x", "1/../x", "1"} {
		if _, _, _, err := s.restoreSource("stardew", "p", token, s.trash); err == nil || err.Error() != "invalid restore token" {
			t.Errorf("restoreSource(%q) err = %v, want invalid restore token", token, err)
		}
	}
}
