package settings

// RecordLastSweep stores the game version last used for a patch-day sweep of gameID and the version of its loader.
func (s *Store) RecordLastSweep(gameID, gameVer, loaderID, loaderVer string) error {
	if gameID == "" {
		return nil
	}
	_, err := s.Update(func(cur *Settings) {
		g := cur.GamePrefs(gameID)
		g.LastSweepGameVersion = gameVer
		putGame(cur, gameID, g)
		if loaderID != "" {
			setLoaderPrefs(cur, loaderID, func(p *LoaderPrefs) { p.LastSweepVersion = loaderVer })
		}
	})
	return err
}

// RecordDownloadsSeen stores the newest download-folder archive mtime (ms) already offered for gameID.
func (s *Store) RecordDownloadsSeen(gameID string, mtime int64) error {
	_, err := s.Update(func(cur *Settings) {
		g := cur.GamePrefs(gameID)
		g.LastDownloadsSeen = mtime
		putGame(cur, gameID, g)
	})
	return err
}
