package queue

import (
	"context"
	"errors"

	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

func fileCategory(ctx context.Context, c *nexus.Client, it Item) string {
	if it.FileID == 0 || c == nil {
		return ""
	}
	files, err := c.Files(ctx, it.ModID)
	if err != nil {
		return ""
	}
	for _, f := range files {
		if f.FileID == it.FileID {
			return f.Category
		}
	}
	return ""
}

func nexusSource(it Item, mod nexus.Mod) profile.Source {
	pic, end := mod.PictureURL, mod.EndorsementCount
	if pic == "" {
		pic = it.picture
	}
	if end == 0 {
		end = it.endorsed
	}
	return profile.Source{
		Kind: profile.KindNexus, Name: it.FileName, ModID: it.ModID, FileID: it.FileID, Version: it.Version,
		Picture: pic, EndorsementCount: end,
	}
}

func (s *Service) installNexusPath(it Item, path string, mod nexus.Mod) error {
	src := nexusSource(it, mod)
	var res profile.InstallResult
	var err error
	if it.MergeAdd && it.Merge != nil && s.d.InstallExtra != nil {
		res, err = s.d.InstallExtra(it.Game, it.Profile, it.Merge.EntryKey, path, src)
	} else {
		res, err = s.d.Install(it.Game, it.Profile, path, src)
	}
	var dup *profile.DuplicateError
	if err == nil || errors.As(err, &dup) {
		dropDownload(path)
	}
	return s.afterInstall(it.ID, res, err, false)
}

func (s *Service) installReadyZip(it Item) (bool, error) {
	if !it.readyZip || it.Repo != "" {
		return false, nil
	}
	path := destPath(s.d.Dir, it.ID, it.FileName)
	return true, s.installNexusPath(it, path, nexus.Mod{PictureURL: "", EndorsementCount: 0})
}
