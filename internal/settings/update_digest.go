package settings

import "slices"

const (
	UpdateDigestOff   = "off"
	UpdateDigestEach  = "each"
	UpdateDigestDaily = "daily"
)

var updateDigestValues = []string{UpdateDigestOff, UpdateDigestEach, UpdateDigestDaily}

func normalizeUpdateDigest(s *Settings) {
	if !slices.Contains(updateDigestValues, s.UpdateDigest) {
		s.UpdateDigest = UpdateDigestDaily
	}
}
