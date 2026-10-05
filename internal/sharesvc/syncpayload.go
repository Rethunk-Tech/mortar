package sharesvc

import "context"

// PreviewPayload resolves a .mortar payload held in memory against the profile profileID ("" for none), for a caller
// that is not the import dialog.
//
//wails:ignore
func (s *Service) PreviewPayload(ctx context.Context, game, profileID string, data []byte) (Preview, error) {
	return s.previewBytes(ctx, game, data, profileID)
}
