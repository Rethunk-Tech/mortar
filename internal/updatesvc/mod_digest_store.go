package updatesvc

import "github.com/Rethunk-AI/mortar/internal/settings"

// SettingsDigestStore reads and writes mod-update digest fields through settings.Service.
type SettingsDigestStore struct {
	Svc *settings.Service
}

func (s SettingsDigestStore) UpdateDigestMode() string {
	return s.Svc.Get().UpdateDigest
}

func (s SettingsDigestStore) LastModUpdateDigest() []string {
	return s.Svc.Get().LastModUpdateDigest
}

func (s SettingsDigestStore) LastModUpdateDigestAt() string {
	return s.Svc.Get().LastModUpdateDigestAt
}

func (s SettingsDigestStore) SaveModUpdateDigest(keys []string, at string) error {
	return s.Svc.SetLastModUpdateDigest(keys, at)
}
