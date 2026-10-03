package control

import (
	"fmt"
	"strconv"

	"github.com/Rethunk-AI/mortar/internal/dlwatch"
)

func (s *Services) downloads() ([]dlwatch.Item, error) {
	if s.Downloads == nil {
		return nil, fmt.Errorf("download watch is not running")
	}
	return s.Downloads.List(), nil
}

func (s *Services) downloadsInstall(p Params) (InstallOutcome, error) {
	if s.Downloads == nil {
		return InstallOutcome{}, fmt.Errorf("download watch is not running")
	}
	n, err := strconv.Atoi(p.Name)
	if err != nil || n < 1 {
		return InstallOutcome{}, fmt.Errorf("name a list number")
	}
	list := s.Downloads.List()
	if n > len(list) {
		return InstallOutcome{}, fmt.Errorf("no download %d", n)
	}
	it := list[n-1]
	res, err := s.Downloads.Install(n)
	if err != nil {
		return InstallOutcome{}, err
	}
	out := InstallOutcome{Added: res.Added, Updated: res.Updated, VersionChanged: res.VersionChanged}
	switch {
	case res.Fomod != nil:
		out.Needs = "fomod"
	case res.Remap != nil:
		out.Needs = "folder"
	}
	if out.Needs != "" && s.Emit != nil {
		s.Emit(InstallAskEvent, InstallAsk{Game: it.Game, Profile: it.ProfileID, Fomod: res.Fomod, Remap: res.Remap})
	}
	if out.Added == nil {
		out.Added = []string{}
	}
	if it.Game != "" && s.Emit != nil {
		s.Emit(ChangedEvent, it.Game)
	}
	return out, nil
}
