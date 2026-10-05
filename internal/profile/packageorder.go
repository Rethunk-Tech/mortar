package profile

import (
	"errors"

	"github.com/Rethunk-Tech/mortar/internal/deploy"
)

// MovePackage moves the entry up (delta < 0, lower priority) or down (delta > 0, wins files) past the next entry
// that deploys, which is the order DeployInputs hands the deployer.
func (s *Store) MovePackage(game, id, key string, delta int) (Profile, error) {
	if delta != -1 && delta != 1 {
		return Profile{}, errors.New("a package moves one place at a time")
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		i := entryIndex(p.Entries, key)
		if i < 0 {
			return errors.New("no such mod in this profile")
		}
		for j := i + delta; j >= 0 && j < len(p.Entries); j += delta {
			if !p.Entries[j].IsOverlay() {
				p.Entries[i], p.Entries[j] = p.Entries[j], p.Entries[i]
				return nil
			}
		}
		return nil
	})
}

// PackageOverrides counts, for each entry key, the files it wins over an earlier package. A game that is
// redirected deploys nothing and has none.
func (s *Store) PackageOverrides(game, id string) (map[string]int, error) {
	in, err := s.DeployInputs(game, id, "")
	if err != nil || len(in.Packages) == 0 {
		return map[string]int{}, err
	}
	d, ok := deploy.Get("link-into-install")
	if !ok {
		return nil, errors.New("no deployer link-into-install")
	}
	plan, err := d.Plan(deploy.View{}, deploy.InstallView{Targets: in.Targets}, in.Packages, nil)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, c := range plan.Conflicts {
		for _, w := range c.Winners {
			out[w]++
		}
	}
	return out, nil
}
