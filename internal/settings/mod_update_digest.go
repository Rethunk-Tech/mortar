package settings

// SetLastModUpdateDigest stores the mod-update digest set and optional toast time.
//
//wails:ignore
func (s *Service) SetLastModUpdateDigest(keys []string, at string) error {
	cp := append([]string(nil), keys...)
	return s.set(func(v *Settings) {
		v.LastModUpdateDigest = cp
		if at != "" {
			v.LastModUpdateDigestAt = at
		}
	})
}
