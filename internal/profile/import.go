package profile

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-smapi"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
	"github.com/Rethunk-Tech/mortar/internal/usererr"

	"github.com/Rethunk-Tech/go-semverish"
)

const importedProfileName = "Imported mods"

const (
	outcomeImported = "imported"
	outcomeSkipped  = "skipped"
	outcomeFailed   = "failed"
)

const configFileName = "config.json"

// GameModPreview is one row PreviewGameMods shows: a mod to copy, or a skipped or failed folder.
type GameModPreview struct {
	ID         mod.ID `json:"id,omitempty"`
	Name       string `json:"name"`
	Version    string `json:"version,omitempty"`
	Source     string `json:"source,omitempty"`
	NexusModID int    `json:"nexusModID,omitempty"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	// Code and Params word a Reason Mortar wrote (not an error's text) in the user's language.
	Code     string            `json:"code,omitempty"`
	Params   map[string]string `json:"params,omitempty"`
	Disabled bool              `json:"disabled,omitempty"`
	// Folder is the absolute path of the top-level folder holding this mod.
	Folder string `json:"folder,omitempty"`
	// Broken is why the game's known-broken list flags this version, empty when it does not, and BrokenKind the
	// entry's kind (components.BrokenKindBroken, ...).
	Broken     string `json:"broken,omitempty"`
	BrokenKind string `json:"brokenKind,omitempty"`
}

// GameModsPreview is the list PreviewGameMods returns, including skips and failures.
type GameModsPreview struct {
	Mods []GameModPreview `json:"mods"`
	// CanMove is whether ImportGameMods can also take the mods out of the game's folder: a game that reads loose
	// files there would otherwise go on loading the originals whatever the profile says.
	CanMove bool `json:"canMove,omitempty"`
}

// GameModOutcome is one top-level folder after ImportGameMods.
type GameModOutcome struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	// Code and Params word a Reason Mortar wrote (not an error's text) in the user's language; Reason stays the English
	// line the CLI prints.
	Code   string            `json:"code,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

// GameModsResult is the profile ImportGameMods created and per-folder results.
type GameModsResult struct {
	Profile  Profile          `json:"profile"`
	Outcomes []GameModOutcome `json:"outcomes"`
	Imported int              `json:"imported"`
	Skipped  int              `json:"skipped"`
	Failed   int              `json:"failed"`
}

// ExternalMod is one folder selected from an external mod manager's profile.
type ExternalMod struct {
	SourcePath string `json:"sourcePath"`
	ID         mod.ID `json:"id"`
	Enabled    bool   `json:"enabled"`
}

type gameModFolder struct {
	dir      string
	label    string
	disabled bool
	mods     []smapimanifest.Mod
	source   Source
	// loose marks a folder game's mod file or folder, which has no manifest: dir may then be a file.
	loose bool
}

type gameModSlot struct {
	folder     gameModFolder
	ready      bool
	outcome    GameModOutcome
	hasOutcome bool
}

func bundledFolder(name string) string {
	switch strings.ToLower(strings.TrimLeft(name, ".")) {
	case "consolecommands":
		return "SMAPI's bundled Console Commands"
	case "savebackup":
		return "SMAPI's bundled Save Backup"
	case "mortarsmapibridge":
		return "Mortar's SMAPI bridge"
	}
	return ""
}

func sourceFromMods(mods []smapimanifest.Mod, folder string) Source {
	for _, m := range mods {
		for _, k := range m.UpdateKeys {
			if n, ok := smapi.NexusUpdateKey(k); ok && n > 0 {
				return Source{Kind: KindNexus, Name: folder, ModID: n, Version: m.Version}
			}
		}
	}
	for _, m := range mods {
		for _, k := range m.UpdateKeys {
			if repo, ok := smapi.GitHubUpdateKey(k); ok {
				return Source{Kind: KindGitHub, Name: folder, Repo: repo, Version: m.Version}
			}
		}
	}
	return Source{Kind: KindLocal, Name: folder}
}

func sourceLabel(s Source) string {
	switch s.Kind {
	case KindNexus:
		return "Nexus"
	case KindGitHub:
		return "GitHub"
	default:
		return "Local"
	}
}

func readTopManifest(dir string) (smapimanifest.Manifest, bool, error) {
	b, err := smapimanifest.ReadFile(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return smapimanifest.Manifest{}, false, nil
	}
	if err != nil {
		return smapimanifest.Manifest{}, true, err
	}
	m, err := smapimanifest.Parse(b)
	return m, true, err
}

var vortexTags = []string{"__folder_managed_by_vortex", ".__folder_managed_by_vortex", "__delete_if_empty"}

// vortexLeftover reports a folder holding no files but Vortex's tag files (what an undeployed mod leaves) or nothing at all.
func vortexLeftover(dir string) bool {
	leftover := true
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || (!d.IsDir() && !slices.ContainsFunc(vortexTags, func(t string) bool { return strings.EqualFold(t, d.Name()) })) {
			leftover = false
			return fs.SkipAll
		}
		return nil
	})
	return leftover
}

func classifyFolder(dir, name string) (gameModSlot, bool) {
	label := strings.TrimLeft(name, ".")
	if bundledFolder(name) != "" || strings.EqualFold(label, "smapi-internal") || bridge.IsOwnListing(label) {
		return gameModSlot{}, false
	}
	failedCode := func(reason, code string, params map[string]string) (gameModSlot, bool) {
		return gameModSlot{
			folder:     gameModFolder{dir: dir},
			outcome:    GameModOutcome{Name: label, Status: outcomeFailed, Reason: reason, Code: code, Params: params},
			hasOutcome: true,
		}, true
	}
	failed := func(reason string) (gameModSlot, bool) { return failedCode(reason, "", nil) }
	m, hasManifest, err := readTopManifest(dir)
	if err != nil && hasManifest {
		return failed(smapimanifest.Invalid{Folder: ".", Err: err}.Error())
	}
	var mods []smapimanifest.Mod
	if hasManifest {
		mods = []smapimanifest.Mod{{Manifest: m, Folder: "."}}
	} else {
		found, invalid, scanErr := smapimanifest.ScanInvalid(dir)
		if scanErr != nil {
			return failed(scanErr.Error())
		}
		if len(found) == 0 {
			if invalid != nil {
				return failed(invalid.Error())
			}
			if vortexLeftover(dir) {
				return gameModSlot{}, false
			}
			return failedCode("No SMAPI mod was found", "noSmapiMod", nil)
		}
		mods = found
	}
	allBundled := true
	for _, im := range mods {
		if !smapimanifest.LoaderManaged(im.ModID()) {
			allBundled = false
			break
		}
	}
	if allBundled {
		return gameModSlot{}, false
	}
	return gameModSlot{
		folder: gameModFolder{
			dir: dir, label: label, disabled: strings.HasPrefix(name, "."),
			mods: mods, source: sourceFromMods(mods, label),
		},
		ready: true,
	}, true
}

func modVersion(f gameModFolder, id mod.ID) string {
	for _, m := range f.mods {
		if mod.Equal(m.ModID(), id) {
			return m.Version
		}
	}
	return ""
}

func preferFolder(a, b gameModFolder, id mod.ID) bool {
	if a.disabled != b.disabled {
		return !a.disabled
	}
	va, vb := modVersion(a, id), modVersion(b, id)
	if c, ok := semverish.Compare(va, vb); ok && c != 0 {
		return c > 0
	}
	return false
}

func resolveDuplicates(slots []gameModSlot) {
	winner := map[string]int{}
	for i, s := range slots {
		if !s.ready {
			continue
		}
		for _, m := range s.folder.mods {
			id := m.ModID().Fold()
			j, ok := winner[id]
			if !ok || preferFolder(s.folder, slots[j].folder, m.ModID()) {
				winner[id] = i
			}
		}
	}
	type skip struct {
		i      int
		reason string
		params map[string]string
	}
	var losers []skip
	seen := map[int]bool{}
	for i, s := range slots {
		if !s.ready {
			continue
		}
		for _, m := range s.folder.mods {
			id := m.ModID().Fold()
			w := winner[id]
			if w == i || seen[i] {
				continue
			}
			seen[i] = true
			losers = append(losers, skip{i, fmt.Sprintf("same mod as %s", slots[w].folder.label), map[string]string{"name": slots[w].folder.label}})
		}
	}
	for _, l := range losers {
		slots[l.i].ready = false
		slots[l.i].hasOutcome = true
		slots[l.i].outcome = GameModOutcome{Name: slots[l.i].folder.label, Status: outcomeSkipped, Reason: l.reason, Code: "sameMod", Params: l.params}
	}
}

// scanModsOf lists what Import from the game's Mods folder would take from modsDir, as the game keeps its mods.
func scanModsOf(game, modsDir string) ([]gameModSlot, error) {
	if kinds, ok := looseModKinds(game); ok {
		return scanLooseMods(modsDir, kinds)
	}
	return scanGameMods(modsDir)
}

func scanGameMods(modsDir string) ([]gameModSlot, error) {
	entries, root, err := readModsDir(modsDir)
	if err != nil {
		return nil, err
	}
	var slots []gameModSlot
	for _, it := range entries {
		child := filepath.Join(modsDir, it.Name())
		if !safefs.RealDirUnder(root, child) {
			continue
		}
		slot, keep := classifyFolder(child, it.Name())
		if !keep {
			continue
		}
		slots = append(slots, slot)
	}
	resolveDuplicates(slots)
	return slots, nil
}

// readModsDir lists a game's mods folder with its real path, so a child can be checked to sit inside it; a missing
// folder lists nothing.
func readModsDir(modsDir string) (entries []os.DirEntry, root string, err error) {
	entries, err = os.ReadDir(modsDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	root, err = safefs.EvalSymlinks(modsDir)
	return entries, root, err
}

func previewMod(f gameModFolder, m smapimanifest.Mod) GameModPreview {
	name := m.Name
	name = cmp.Or(name, m.ModID().Local())
	return GameModPreview{
		ID: m.ModID(), Name: name, Version: m.Version, Source: sourceLabel(f.source),
		NexusModID: f.source.ModID, Status: outcomeImported, Disabled: f.disabled, Folder: f.dir,
	}
}

func previewFrom(slots []gameModSlot) GameModsPreview {
	var mods []GameModPreview
	for _, s := range slots {
		if s.ready && s.folder.loose {
			mods = append(mods, GameModPreview{Name: s.folder.label, Status: outcomeImported, Disabled: s.folder.disabled, Folder: s.folder.dir})
			continue
		}
		if s.ready {
			for _, m := range s.folder.mods {
				mods = append(mods, previewMod(s.folder, m))
			}
			continue
		}
		if s.hasOutcome {
			mods = append(mods, GameModPreview{
				Name: s.outcome.Name, Status: s.outcome.Status, Reason: s.outcome.Reason,
				Code: s.outcome.Code, Params: s.outcome.Params,
				Disabled: s.folder.disabled,
			})
		}
	}
	if mods == nil {
		mods = []GameModPreview{}
	}
	return GameModsPreview{Mods: mods}
}

func outcomesFrom(slots []gameModSlot) []GameModOutcome {
	var out []GameModOutcome
	for _, s := range slots {
		if s.hasOutcome {
			out = append(out, s.outcome)
		}
	}
	return out
}

// PreviewGameMods lists mods that ImportGameMods would copy from modsDir. It reads only.
func (s *Store) PreviewGameMods(game, modsDir string) (GameModsPreview, error) {
	slots, err := scanModsOf(game, modsDir)
	if err != nil {
		return GameModsPreview{}, err
	}
	pv := previewFrom(slots)
	_, pv.CanMove = looseModKinds(game)
	return pv, nil
}

func carryConfig(srcRoot, destDir, folder string) error {
	rel := folder
	if rel == "." {
		rel = ""
	}
	from := filepath.Join(srcRoot, filepath.FromSlash(rel), configFileName)
	b, err := safefs.ReadFile(from)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return safefs.WriteFile(filepath.Join(destDir, configFileName), b, 0o600)
}

// addResolvingLinks stores dir with every file link replaced by the file it points to, so a mod another manager deployed
// as links into its own staging folder imports as plain files; the store itself still refuses links that leave a folder.
func (s *Store) addResolvingLinks(ctx context.Context, game, dir string) (string, error) {
	scratch, err := os.MkdirTemp(filepath.Dir(s.root), "mortar-import-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = safefs.RemoveAll(scratch) }()
	copied := filepath.Join(scratch, "mod")
	if err := datadir.CopyTreeResolvingLinks(dir, copied); err != nil {
		return "", err
	}
	return s.items.AddHashedDir(ctx, game, copied)
}

// importFolder adds f to the profile and returns its store key and the history event the last of its changes recorded.
func (s *Store) importFolder(ctx context.Context, game, id string, f gameModFolder) (key, change string, err error) {
	if f.loose {
		return s.importLoose(ctx, game, id, f)
	}
	key, err = s.addResolvingLinks(ctx, game, f.dir)
	if err != nil {
		return "", "", err
	}
	added, err := s.AddEntry(game, id, key, f.source)
	if err != nil {
		return "", "", err
	}
	change = added.LastChange
	if f.disabled {
		for _, m := range f.mods {
			disabled, err := s.SetModEnabled(game, id, key, m.ModID(), false)
			if err != nil {
				return "", "", err
			}
			change = disabled.LastChange
		}
	}
	for _, m := range f.mods {
		dest, err := s.ModFolder(game, id, key, m.ModID())
		if err != nil {
			return "", "", err
		}
		if err := carryConfig(f.dir, dest, m.Folder); err != nil {
			return "", "", err
		}
	}
	return key, change, nil
}

// ImportGameMods copies each importable folder under modsDir into the store and a new "Imported mods" profile.
// It never writes to modsDir, unless move is set for a game that reads loose files there (GameModsPreview.CanMove):
// each mod that came in is then removed from modsDir, as MoveGameMods removes a folder.
func (s *Store) ImportGameMods(ctx context.Context, game, modsDir string, move bool) (GameModsResult, error) {
	if _, loose := looseModKinds(game); !loose {
		move = false
	}
	if move && s.GameRunning != nil && s.GameRunning(game) {
		return GameModsResult{}, usererr.Wrap(usererr.Busy, errors.New("the game is running: close it before moving mods out of its Mods folder"))
	}
	slots, err := scanModsOf(game, modsDir)
	if err != nil {
		return GameModsResult{}, err
	}
	existing, err := s.listOK(game)
	if err != nil {
		return GameModsResult{}, err
	}
	taken := make([]string, len(existing))
	for i, p := range existing {
		taken[i] = p.Name
	}
	created, err := s.Create(game, UniqueName(taken, importedProfileName))
	if err != nil {
		return GameModsResult{}, err
	}
	created, err = s.SetOrigin(game, created.ID, OriginGameMods, "")
	if err != nil {
		return GameModsResult{}, errors.Join(err, s.Delete(game, created.ID))
	}
	s.setHistoryQuiet(created.ID, true)
	defer s.setHistoryQuiet(created.ID, false)
	res := GameModsResult{Profile: created, Outcomes: outcomesFrom(slots)}
	for _, o := range res.Outcomes {
		switch o.Status {
		case outcomeSkipped:
			res.Skipped++
		case outcomeFailed:
			res.Failed++
		}
	}
	total := 0
	for _, slot := range slots {
		if slot.ready {
			total++
		}
	}
	done := 0
	for _, slot := range slots {
		if !slot.ready {
			continue
		}
		s.reportGameModsProgress(game, created.ID, done, total, slot.folder.label)
		done++
		outcome := GameModOutcome{Name: slot.folder.label, Status: outcomeImported}
		if _, _, err := s.importFolder(ctx, game, created.ID, slot.folder); err != nil {
			outcome.Status, outcome.Reason = outcomeFailed, err.Error()
			if ie, ok := errors.AsType[*InstallError](err); ok {
				outcome.Reason = ie.Msg
			}
			res.Failed++
		} else {
			res.Imported++
			if move {
				if err := s.removeFromGameMods(game, created.ID, slot.folder.dir); err != nil {
					outcome.Reason = "added, but it could not be removed from the game's Mods folder: " + err.Error()
					outcome.Code, outcome.Params = "notRemoved", map[string]string{"error": err.Error()}
				}
			}
		}
		res.Outcomes = append(res.Outcomes, outcome)
	}
	p, err := s.read(game, created.ID)
	if err != nil {
		return res, err
	}
	res.Profile = p
	if res.Outcomes == nil {
		res.Outcomes = []GameModOutcome{}
	}
	if res.Imported > 0 {
		kind := ChangeImported
		if move {
			kind = ChangeMoved
		}
		if res.Profile.LastChange, err = s.recordSnapshot(game, created.ID, historyImported, HistoryEvent{Change: kind}, res.Imported); err != nil {
			return res, err
		}
		if err := s.RecordModsSnapshot(game, created.ID); err != nil {
			return res, err
		}
	}
	return res, nil
}

// ImportExternalMods copies the selected external-manager folders into an existing profile and returns the history
// events the import recorded, oldest first; under a history batch that is the batch's one event.
func (s *Store) ImportExternalMods(ctx context.Context, game, id string, mods []ExternalMod) ([]string, error) {
	enabled := make(map[string]bool, len(mods))
	paths := make(map[string]bool, len(mods))
	for _, im := range mods {
		if im.SourcePath == "" {
			continue
		}
		paths[filepath.Clean(im.SourcePath)] = true
		enabled[im.ID.Fold()] = im.Enabled
	}
	var slots []gameModSlot
	for path := range paths {
		name := filepath.Base(path)
		slot, keep := classifyFolder(path, name)
		if keep {
			slots = append(slots, slot)
		}
	}
	resolveDuplicates(slots)
	var changes []string
	for _, slot := range slots {
		if !slot.ready {
			continue
		}
		key, change, err := s.importFolder(ctx, game, id, slot.folder)
		if err != nil {
			return changes, err
		}
		changes = append(changes, change)
		for _, im := range slot.folder.mods {
			if want, ok := enabled[im.ModID().Fold()]; ok {
				toggled, err := s.SetModEnabled(game, id, key, im.ModID(), want)
				if err != nil {
					return changes, err
				}
				changes = append(changes, toggled.LastChange)
			}
		}
	}
	changes = slices.Compact(slices.DeleteFunc(changes, func(c string) bool { return c == "" }))
	return changes, s.RecordModsSnapshot(game, id)
}
