package curseforge

import (
	"context"
	"strconv"

	cf "github.com/Rethunk-Tech/go-curseforge"
)

// Schemes is the scheme of the site's Install button.
func (Driver) Schemes() []string { return []string{"curseforge"} }

// HandleLinksDefault is false: the CurseForge app owns these links for every game it serves, so Mortar takes them
// from the system only when the user opts in.
func (Driver) HandleLinksDefault() bool { return false }

// Ref names one file of a project in an install link.
type Ref = cf.Ref

// ParseLink reads curseforge://install?addonId=<project>&fileId=<file>.
func ParseLink(raw string) (Ref, error) { return cf.ParseLink(raw) }

// Project is what a link's project is: its name, the site's id of its game, and whether its author forbids
// downloads outside CurseForge's own page.
type Project struct {
	Name       string
	GameID     int
	Restricted bool
}

// Project reads the project a link names.
func (d Driver) Project(ctx context.Context, id int) (Project, error) {
	m, err := d.mod(ctx, strconv.Itoa(id))
	return Project{
		Name: m.Name, GameID: m.GameID, Restricted: m.Restricted(),
	}, err
}

// PageURL is the project's page address, whose last segment is its slug; empty when the site gives none.
func (d Driver) PageURL(ctx context.Context, id string) (string, error) {
	m, err := d.mod(ctx, id)
	return m.Links.WebsiteURL, err
}
