package sharesvc

import (
	"context"
	"errors"
	"net/http"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/sharehost"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Hosted is a profile's short link on mortar.rethunk.tech. The owner token is not included.
type Hosted struct {
	Code    string `json:"code"`
	Version int    `json:"version"`
	URL     string `json:"url"`
}

func (s *Service) records() sharehost.Store {
	return sharehost.Store{Dir: s.d.Dir}
}

func (s *Service) hostedRecord(game, profileID string) *Hosted {
	rec, found, err := s.records().Get(game, profileID)
	if err != nil || !found || rec.Code == "" {
		return nil
	}
	return &Hosted{Code: rec.Code, Version: rec.Version, URL: sharehost.ShortURL(game, rec.Code)}
}

func (s *Service) hostedPayload(game, profileID string, keys []string, include share.Include) (string, error) {
	p, err := s.find(game, profileID)
	if err != nil {
		return "", err
	}
	p = withEntryKeys(p, keys)
	res, err := share.Encode(game, p, s.d.Profiles.ShareFacts(game, p), include)
	if err != nil {
		return "", err
	}
	return res.Payload, nil
}

// CreateHostedShare POSTs the profile's share payload and stores the owner token in the data dir.
func (s *Service) CreateHostedShare(ctx context.Context, game, profileID string, keys []string, include share.Include) (Hosted, error) {
	payload, err := s.hostedPayload(game, profileID, keys, include)
	if err != nil {
		return Hosted{}, err
	}
	created, err := s.host.Create(ctx, payload)
	if err != nil {
		return Hosted{}, hostedErr(err)
	}
	rec := sharehost.Record(created)
	if err := s.records().Put(game, profileID, rec); err != nil {
		return Hosted{}, err
	}
	return Hosted{Code: rec.Code, Version: rec.Version, URL: sharehost.ShortURL(game, rec.Code)}, nil
}

// UpdateHostedShare PUTs a new payload for the profile's existing short link.
func (s *Service) UpdateHostedShare(ctx context.Context, game, profileID string, keys []string, include share.Include) (Hosted, error) {
	rec, found, err := s.records().Get(game, profileID)
	if err != nil {
		return Hosted{}, err
	}
	if !found || rec.Code == "" || rec.Token == "" {
		return Hosted{}, usererr.Wrap(usererr.NotFound, errors.New("this profile has no short link"))
	}
	payload, err := s.hostedPayload(game, profileID, keys, include)
	if err != nil {
		return Hosted{}, err
	}
	upd, err := s.host.Update(ctx, rec.Code, rec.Token, payload)
	if err != nil {
		return Hosted{}, hostedErr(err)
	}
	rec.Version = upd.Version
	if err := s.records().Put(game, profileID, rec); err != nil {
		return Hosted{}, err
	}
	return Hosted{Code: rec.Code, Version: rec.Version, URL: sharehost.ShortURL(game, rec.Code)}, nil
}

func hostedErr(err error) error {
	var se *httpkit.StatusError
	if !errors.As(err, &se) {
		return err
	}
	switch se.Code {
	case http.StatusBadRequest, http.StatusRequestEntityTooLarge:
		return usererr.Wrap(usererr.Invalid, err)
	case http.StatusForbidden:
		return usererr.Wrap(usererr.Permission, errors.New("that short link is not yours"))
	case http.StatusNotFound:
		return usererr.Wrap(usererr.NotFound, err)
	case http.StatusTooManyRequests:
		return usererr.Wrap(usererr.RateLimited, err)
	default:
		return err
	}
}
