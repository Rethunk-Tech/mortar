package settings

// RecordLastSweep stores the game and SMAPI versions last used for a patch-day sweep of gameID.
func (s *Store) RecordLastSweep(gameID, gameVer, smapiVer string) error {
	if gameID == "" {
		return nil
	}
	_, err := s.Update(func(cur *Settings) {
		g := cur.GamePrefs(gameID)
		g.LastSweepGameVersion = gameVer
		g.LastSweepSMAPIVersion = smapiVer
		putGame(cur, gameID, g)
	})
	return err
}
