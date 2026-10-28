package templates

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// RestoreTemplate puts back a template DeleteTemplate removed, replacing one of the same name.
func (s *Service) RestoreTemplate(game string, t Template) error {
	name, err := profile.CleanName("template", t.Name)
	if err != nil {
		return err
	}
	t.Name, t.Game = name, game
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.read(game)
	if err != nil {
		return err
	}
	if i := indexOf(list, name); i >= 0 {
		list[i] = t
	} else {
		list = append(list, t)
	}
	return s.write(game, list)
}

// RenameTemplate gives a template a new name. It fails when another template already has that name.
func (s *Service) RenameTemplate(game, name, newName string) error {
	newName, err := profile.CleanName("template", newName)
	if err != nil {
		return err
	}
	return s.editTemplate(game, name, func(list []Template, i int) ([]Template, error) {
		if j := indexOf(list, newName); j >= 0 && j != i {
			return nil, usererr.Wrap(usererr.Invalid, fmt.Errorf("a template named %q already exists", newName))
		}
		list[i].Name = newName
		return list, nil
	})
}

// editTemplate rewrites the game's templates as edit returns them, given the index of the one named name.
func (s *Service) editTemplate(game, name string, edit func(list []Template, i int) ([]Template, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.read(game)
	if err != nil {
		return err
	}
	i := indexOf(list, name)
	if i < 0 {
		return usererr.Wrap(usererr.NotFound, fmt.Errorf("template %q was not found", name))
	}
	if list, err = edit(list, i); err != nil {
		return err
	}
	return s.write(game, list)
}
