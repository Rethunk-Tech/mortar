package devsvc

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// githubOrigin is the owner and repository of the folder's origin remote when it is on github.com.
func (s *Service) githubOrigin(ctx context.Context, dir string) (owner, repo string, ok bool) {
	out, err := s.run(ctx, dir, "git", "remote", "get-url", "origin")
	if err != nil {
		return "", "", false
	}
	owner, repo, err = ParseRepo(string(out))
	return owner, repo, err == nil && gh.ValidRepo(owner+"/"+repo)
}

// UploadGitHub creates the GitHub release for the project's tag v<version>, with that version's CHANGELOG notes, and
// attaches the newest package. A tag that already has a release gets the package as one more asset, unless it has it.
// It returns the release's page.
func (s *Service) UploadGitHub(ctx context.Context, gameID, dir, version string) (string, error) {
	var owner, repo string
	p, zipPath, notes, err := s.prepareUpload(gameID, dir, version, true, func(p settings.DevProject) error {
		var ok bool
		if owner, repo, ok = s.githubOrigin(ctx, p.Dir); !ok {
			return usererr.Wrap(usererr.Invalid, errors.New("the project has no github.com origin"))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	tag := "v" + version
	if !gh.DefaultAuth.LoggedIn(ctx) {
		return "", usererr.Wrap(usererr.Permission, errors.New("sign in to GitHub with the gh CLI: run gh auth login"))
	}
	out, err := s.run(ctx, p.Dir, "git", "ls-remote", "--tags", "origin", "refs/tags/"+tag)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(out)) == "" {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("push the tag %s to GitHub first", tag))
	}
	c := &gh.Client{HTTP: s.httpClient(), APIBase: s.GitHubURL}
	rel, found, err := c.ReleaseByTag(ctx, owner, repo, tag)
	if err == nil && !found {
		rel, err = c.CreateRelease(ctx, owner, repo, gh.NewRelease{Tag: tag, Name: tag, Body: notes})
	}
	if err != nil {
		return "", githubError(err)
	}
	name := filepath.Base(zipPath)
	for _, a := range rel.Assets {
		if a.Name == name {
			return rel.HTMLURL, nil
		}
	}
	if _, err := c.UploadAsset(ctx, rel, zipPath, "application/zip"); err != nil {
		return "", githubError(err)
	}
	return rel.HTMLURL, nil
}

func githubError(err error) error {
	if errors.Is(err, gh.ErrNoToken) {
		return usererr.Wrap(usererr.Permission, errors.New("sign in to GitHub with the gh CLI: run gh auth login"))
	}
	return err
}
