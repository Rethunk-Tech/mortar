package queue

import (
	"cmp"
	"context"
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// pageFiles lists the item's mod page files, nil when they cannot be fetched.
func pageFiles(ctx context.Context, c *nexus.Client, it Item) []nexus.File {
	if it.FileID == 0 || c == nil {
		return nil
	}
	t, err := game.NexusTitle(it.Game)
	if err != nil {
		return nil
	}
	files, err := c.Files(ctx, t, it.ModID)
	if err != nil {
		return nil
	}
	return files
}

func fileCategoryOf(files []nexus.File, fileID int) string {
	for _, f := range files {
		if f.FileID == fileID {
			return f.Category
		}
	}
	return ""
}

// archiveModIDs lists the mods an archive holds, nil when its listing cannot say: a FOMOD installs only some of them.
func archiveModIDs(path string) []mod.ID {
	p, err := archive.PreviewArchive(path)
	if err != nil || p.Truncated || p.Fomod {
		return nil
	}
	var ids []mod.ID
	for _, m := range p.Manifests {
		ids = append(ids, m.ID)
	}
	return ids
}

func (it Item) incoming(category string, files []nexus.File, ids []mod.ID) profile.IncomingFile {
	return profile.IncomingFile{ModID: it.ModID, FileID: it.FileID, Category: category, Files: files, ModIDs: ids}
}

func nexusSource(it Item, im nexus.Mod) profile.Source {
	pic, end := im.PictureURL, im.EndorsementCount
	pic = cmp.Or(pic, it.Picture)
	end = cmp.Or(end, it.endorsed)
	return sourceWithOptions(it, profile.Source{
		Kind: profile.KindNexus, Name: it.FileName, ModID: it.ModID, FileID: it.FileID, Version: it.Version,
		Picture: pic, EndorsementCount: end, ModName: it.Name, Category: it.Category,
	})
}

func sourceWithOptions(it Item, source profile.Source) profile.Source {
	choices := it.fomod
	if choices == nil {
		choices = it.Fomod
	}
	if len(choices) > 0 {
		source = source.WithFomod(choices)
	}
	if len(it.Disabled) > 0 {
		source = source.WithDisabled(it.Disabled)
	}
	if it.Current != 0 {
		source = source.WithReplacing(it.Current)
	}
	if it.Overlay != nil {
		source = source.WithOverlay(it.Overlay.From, it.Overlay.To).WithOverlayOff(it.Overlay.Off)
	}
	return source
}

func (s *Service) installNexusPath(ctx context.Context, it Item, path string, im nexus.Mod) error {
	src := nexusSource(it, im)
	var res profile.InstallResult
	var err error
	s.installMu.Lock()
	defer s.installMu.Unlock()
	if it.MergeAdd && it.Merge != nil && s.d.InstallExtra != nil {
		res, err = s.d.InstallExtra(ctx, it.Game, it.Profile, it.Merge.EntryKey, path, src)
	} else {
		res, err = s.d.Install(ctx, it.Game, it.Profile, path, src)
	}
	if err == nil && src.ModID > 0 && s.d.Track != nil {
		s.d.Track(ctx, it.Game, src.ModID)
	}
	var dup *profile.DuplicateError
	if err == nil || errors.As(err, &dup) {
		s.dropDownloadUnlessKept(path)
	}
	return s.afterInstall(it.ID, res, err, false)
}

const rawXNBMessage = "This file replaces game files directly (raw .xnb). Mortar installs SMAPI mods; use the mod's Content Patcher version."

func (s *Service) contentPatcherHint(ctx context.Context, it Item, err error) error {
	var installErr *profile.InstallError
	if !errors.As(err, &installErr) || !strings.HasPrefix(installErr.Msg, rawXNBMessage) {
		return err
	}
	c, clientErr := s.d.Client()
	if clientErr != nil {
		return err
	}
	t, titleErr := game.NexusTitle(it.Game)
	if titleErr != nil {
		return titleErr
	}
	files, filesErr := c.Files(ctx, t, it.ModID)
	if filesErr != nil {
		return err
	}
	var name string
	bestID := 0
	for _, file := range files {
		if file.FileID == it.FileID || file.FileID <= bestID {
			continue
		}
		if !strings.Contains(strings.ToLower(file.Name+" "+file.FileName), "content patcher") {
			continue
		}
		name, bestID = file.FileName, file.FileID
	}
	if name == "" {
		return err
	}
	return &profile.InstallError{Msg: installErr.Msg + " Content Patcher file: " + name, Err: err}
}

func (s *Service) installReadyZip(ctx context.Context, it Item) (bool, error) {
	if !it.readyZip || it.Repo != "" {
		return false, nil
	}
	path := s.dest(it.ID, it.FileName)
	return true, s.installNexusPath(ctx, it, path, nexus.Mod{PictureURL: "", EndorsementCount: 0})
}
