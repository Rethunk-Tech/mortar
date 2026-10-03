package gmcm

type ProfileDirs interface {
	ProfileDir(game, profile string) (string, error)
}

type Service struct {
	Dirs ProfileDirs
}

func (s *Service) dir(game, profile string) (string, error) {
	if s == nil || s.Dirs == nil {
		return "", ErrNoCapture
	}
	return s.Dirs.ProfileDir(game, profile)
}

func (s *Service) GmcmMenu(game, profile, uniqueID string) (Capture, error) {
	d, err := s.dir(game, profile)
	if err != nil {
		return Capture{}, err
	}
	return ReadCapture(d, uniqueID)
}

func (s *Service) PendingGmcm(game, profile, uniqueID string) (Pending, error) {
	d, err := s.dir(game, profile)
	if err != nil {
		return Pending{}, err
	}
	return ReadPending(d, uniqueID)
}

func (s *Service) SetGmcmEdits(game, profile, uniqueID string, edits []Edit) error {
	d, err := s.dir(game, profile)
	if err != nil {
		return err
	}
	return WritePending(d, uniqueID, edits)
}

func (s *Service) GmcmResult(game, profile, uniqueID string) (Result, error) {
	d, err := s.dir(game, profile)
	if err != nil {
		return Result{}, err
	}
	return ReadResult(d, uniqueID)
}

func (s *Service) CapturedMods(game, profile string) ([]string, error) {
	d, err := s.dir(game, profile)
	if err != nil {
		return nil, err
	}
	return CapturedIDs(d)
}
