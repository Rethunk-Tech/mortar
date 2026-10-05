package syncsvc

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
)

// profileStore is the part of the profile store the sync uses.
type profileStore interface {
	List(game string) ([]profile.Profile, error)
	Create(game, name string) (profile.Profile, error)
}

// Shares is the part of the share service the sync uses: the .mortar export and the import that makes a profile match.
type Shares interface {
	ExportBytes(game, profileID string, keys []string) ([]byte, []string, error)
	PreviewPayload(ctx context.Context, game, profileID string, data []byte) (sharesvc.Preview, error)
	Replace(ctx context.Context, game, session, profileID string, exclude []string) (sharesvc.Result, error)
}

// AppSource is the Source over the app's profiles and share service.
type AppSource struct {
	profiles profileStore
	shares   Shares
}

// NewSource is the Source over the app's profiles and share service.
func NewSource(profiles *profile.Store, shares *sharesvc.Service) *AppSource {
	return &AppSource{profiles: profiles, shares: shares}
}

func (*AppSource) Games() []string { return game.Implemented() }

func (a *AppSource) Profiles(gameID string) ([]Ref, error) {
	all, err := a.profiles.List(gameID)
	if err != nil {
		return nil, err
	}
	out := make([]Ref, 0, len(all))
	for _, p := range all {
		if p.Error == "" {
			out = append(out, Ref{Game: gameID, ID: p.ID, Name: p.Name, Updated: p.Updated})
		}
	}
	return out, nil
}

func (a *AppSource) Export(gameID, id string) ([]byte, error) {
	b, _, err := a.shares.ExportBytes(gameID, id, nil)
	return b, err
}

func (a *AppSource) Create(gameID, name string) (string, error) {
	p, err := a.profiles.Create(gameID, name)
	return p.ID, err
}

func (a *AppSource) Preview(ctx context.Context, gameID, id string, payload []byte) (Diff, error) {
	pv, err := a.shares.PreviewPayload(ctx, gameID, id, payload)
	if err != nil {
		return Diff{}, err
	}
	d := Diff{Add: []string{}, Remove: append([]string{}, pv.Replace.Remove...)}
	for _, m := range pv.Mods {
		if m.State != sharesvc.StateInstalled {
			d.Add = append(d.Add, m.Name)
		}
	}
	return d, nil
}

func (a *AppSource) Apply(ctx context.Context, gameID, id string, payload []byte) error {
	pv, err := a.shares.PreviewPayload(ctx, gameID, id, payload)
	if err != nil {
		return err
	}
	_, err = a.shares.Replace(ctx, gameID, pv.Session, id, nil)
	return err
}
