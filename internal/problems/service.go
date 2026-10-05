package problems

import (
	"context"
	"errors"
	"log"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// Service exposes the problem checks to the frontend.
type Service struct {
	home     string
	settings *settings.Store
	profiles *profile.Store
	meta     Meta
	Runs     RunReader
	// Damage lists the game's store items whose files failed verification; nil means none.
	Damage func(game string) map[string]store.Damage
	// NexusFiles, when set, confirms flagged updates against Nexus's live file lists in one call.
	NexusFiles NexusFilesOf
	// NexusPages gives the page data of many Nexus mods in a few batched requests; nil when signed out or unwired.
	NexusPages NexusPagesOf
	// Throttle waits for a source's turn to be asked (the download queue's per-source limit) and returns the func that
	// ends it; nil asks without waiting.
	Throttle func(ctx context.Context, source string) (release func(), err error)

	mu      sync.Mutex
	drift   map[string]driftScan
	cache   map[string]cached
	updates map[string]cachedUpdates
	checks  map[string]*problemCall
	assets  map[string]cachedIndex
}

type problemCall struct {
	done     chan struct{}
	result   Result
	once     sync.Once
	joined   chan struct{}
	joinOnce sync.Once
}

func (s *Service) shareCheck(ctx context.Context, key string, check func() Result) (Result, error) {
	s.mu.Lock()
	if call, ok := s.checks[key]; ok {
		call.joinOnce.Do(func() { close(call.joined) })
		s.mu.Unlock()
		select {
		case <-call.done:
			return call.result, nil
		case <-ctx.Done():
			return Result{}, ctx.Err()
		}
	}
	call := &problemCall{done: make(chan struct{}), joined: make(chan struct{})}
	s.checks[key] = call
	s.mu.Unlock()

	r := check()
	s.mu.Lock()
	call.once.Do(func() {
		call.result = r
		delete(s.checks, key)
		close(call.done)
	})
	s.mu.Unlock()
	return r, nil
}

type cachedUpdates struct {
	fingerprint string
	at          time.Time
	result      UpdatesResult
}

// cached is a result with the fingerprint of the mods and environment it was computed for.
type cached struct {
	fingerprint string
	result      Result
	// until is when a result with unknown parts expires, so a lookup that failed is retried soon
	// without re-running the whole check on every refresh; zero for complete results.
	until time.Time
	// sameJob is the result's same-job rows; the fingerprint covers their inputs (the mods and the last run).
	sameJob []framework.Redundant
}

const unknownResultTTL = 2 * time.Minute

func (c cached) fresh(fp string, now time.Time) bool {
	return c.fingerprint == fp && (c.until.IsZero() || now.Before(c.until))
}

func NewService(home string, s *settings.Store, profiles *profile.Store, m *meta.Client) *Service {
	return &Service{home: home, settings: s, profiles: profiles, meta: m, drift: map[string]driftScan{}, cache: map[string]cached{}, updates: map[string]cachedUpdates{}, checks: map[string]*problemCall{}, assets: map[string]cachedIndex{}}
}

func platform() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	}
	return "Linux"
}

// Environment reads the versions from the game's logs; without an install they stay empty.
//
//wails:ignore
func (s *Service) Environment(id string) Environment {
	env := Environment{Platform: platform(), VersionScheme: versionScheme(id)}
	env.Nexus, _ = game.NexusTitle(id)
	if game.Find(id) == nil {
		return env
	}
	set := s.settings.Get()
	dir, err := game.InstallDir(s.home, set, id)
	if err != nil || dir == "" {
		return env
	}
	st, _ := game.LoaderStatus(id, "", dir, set.Loaders)
	env.GameVersion, env.APIVersion = st.GameVersion, st.Version
	return env
}

// playerLog is the text of the Unity player log a loader's analyzers read, empty when the loader reads none or the log
// is not there.
func (s *Service) playerLog(gameID string, l loader.Loader) string {
	w, ok := l.(loader.WithPlayerLog)
	if !ok {
		return ""
	}
	path, err := game.PathFor(s.home, s.settings.Get(), gameID, "", w.PlayerLogRole())
	if err != nil {
		return ""
	}
	b, _ := fsx.ReadFile(path)
	return string(b)
}

// versionScheme is the version scheme of the game's loader.
func versionScheme(gameID string) string {
	for _, l := range game.Loaders(gameID) {
		if v, ok := l.(loader.VersionScheme); ok {
			return v.VersionScheme()
		}
	}
	return deps.Opaque
}

// nexusDomain is the Nexus domain of a game, empty when Nexus does not host it.
func nexusDomain(gameID string) string {
	t, _ := game.NexusTitle(gameID)
	return t.Domain
}

func fingerprint(env Environment, mods []framework.Mod, runID string) string {
	var b strings.Builder
	b.WriteString(env.GameVersion + "|" + env.APIVersion + "|run:" + runID)
	for _, m := range mods {
		b.WriteString("\n" + m.Key + "|" + string(m.ModID()) + "|" + m.Version + "|" + m.Name + "|ch:" + m.UpdateChannel)
		b.WriteString("|desc:" + m.Description)
		if m.Enabled {
			b.WriteString("|on")
		}
		if m.Folder != "" {
			if info, err := os.Stat(filepath.Join(m.Folder, "content.json")); err == nil {
				b.WriteString("|" + strconv.FormatInt(info.ModTime().UnixNano(), 10))
			}
			if info, err := os.Stat(filepath.Join(m.Folder, "config.json")); err == nil {
				b.WriteString("|config:" + strconv.FormatInt(info.ModTime().UnixNano(), 10))
			}
		}
		for _, d := range m.Dependencies {
			b.WriteString("|" + string(d.ModID()) + ">=" + d.MinimumVersion)
		}
		for _, id := range m.LoadAfter {
			b.WriteString("|after:" + string(id))
		}
	}
	return b.String()
}

func (s *Service) installed(gameID, id string) ([]framework.Mod, error) {
	installed, err := s.profiles.Installed(gameID, id)
	if err != nil {
		return nil, err
	}
	mods := make([]framework.Mod, len(installed))
	for i, m := range installed {
		mods[i] = framework.Mod{
			Key: m.Key, SourceKind: m.Source.Kind, SourceVersion: m.Source.Version, SourceModID: m.Source.ModID, SourceName: m.Source.Name, SourceRepo: m.Source.Repo, Enabled: m.Enabled, Folder: m.Folder,
			Pinned: m.Pinned, SkipVersion: m.SkipVersion, SkipSources: m.SkipSources, IgnoreUpdates: m.IgnoreUpdates,
			UpdateChannel: m.UpdateChannel,
			LoadAfter:     m.LoadAfter, Manifest: m.Manifest,
		}
	}
	return mods, nil
}

// Problems checks the profile's mods and leaves out each conflict's evidence, which is most of the result and
// only an expanded row shows; ConflictEvidence fetches it.
func (s *Service) Problems(ctx context.Context, gameID, id string) (Result, error) {
	r, err := s.ProblemsWithEvidence(ctx, gameID, id)
	if err != nil {
		return r, err
	}
	r.AssetConflicts = slices.Clone(r.AssetConflicts)
	for i := range r.AssetConflicts {
		r.AssetConflicts[i].Evidence = nil
	}
	r.Dismissed = slices.Clone(r.Dismissed)
	for i, d := range r.Dismissed {
		if d.AssetConflict != nil {
			c := *d.AssetConflict
			c.Evidence = nil
			r.Dismissed[i].AssetConflict = &c
		}
	}
	return r, nil
}

// ConflictEvidence returns the per-pack evidence of one asset conflict, shown or dismissed.
func (s *Service) ConflictEvidence(ctx context.Context, gameID, id, kind, target string) ([]framework.ConflictEvidence, error) {
	r, err := s.ProblemsWithEvidence(ctx, gameID, id)
	if err != nil {
		return nil, err
	}
	for _, c := range r.AssetConflicts {
		if c.Kind == kind && c.Target == target {
			return c.Evidence, nil
		}
	}
	for _, d := range r.Dismissed {
		if c := d.AssetConflict; c != nil && c.Kind == kind && c.Target == target {
			return c.Evidence, nil
		}
	}
	return []framework.ConflictEvidence{}, nil
}

// ProblemsWithEvidence is Problems with every conflict's evidence. The answer is kept until the mods or versions
// change, unless a lookup failed, in which case the next call tries again.
//
//wails:ignore
func (s *Service) ProblemsWithEvidence(ctx context.Context, gameID, id string) (Result, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return Result{}, err
	}
	env := s.Environment(gameID)
	runID := ""
	if s.Runs != nil {
		lastID, err := s.Runs.LastRunID(gameID, id)
		if err == nil {
			runID = lastID
		}
	}
	fp := fingerprint(env, mods, runID)
	depth := settings.ConflictScanFull
	if s.settings != nil {
		depth = settings.ResolveAt(s.settings.Get(), "conflictScanDepth", settings.Scope{Game: gameID}, nil)
	}
	fp += "|scan:" + depth
	pkgs, _ := s.profiles.EnabledPackages(gameID, id)
	pkgKeys := make([]string, len(pkgs))
	for i, p := range pkgs {
		pkgKeys[i] = p.Key
	}
	fp += "|pkgs:" + strings.Join(pkgKeys, ",")
	key := gameID + "/" + id
	s.mu.Lock()
	c, ok := s.cache[key]
	s.mu.Unlock()
	if ok && c.fresh(fp, time.Now()) {
		return s.withDrift(gameID, id, fp, s.withDismissed(gameID, id, s.withCompat(ctx, c.result, gameID, mods, c.sameJob)))
	}

	checkKey := key + "\x00" + fp
	s.mu.Lock()
	if c, ok := s.cache[key]; ok && c.fresh(fp, time.Now()) {
		s.mu.Unlock()
		return s.withDrift(gameID, id, fp, s.withDismissed(gameID, id, s.withCompat(ctx, c.result, gameID, mods, c.sameJob)))
	}
	s.mu.Unlock()

	r, err := s.shareCheck(ctx, checkKey, func() Result {
		contentpatcher.SkipImageOverlap = depth == settings.ConflictScanSkipImages
		defer func() { contentpatcher.SkipImageOverlap = false }()
		r := Check(ctx, s.metaFor(gameID), env, mods)
		r.Broken = append(r.Broken, authorMarkedMods(s.home, env.Nexus.Domain, slices.DeleteFunc(slices.Clone(mods), func(x framework.Mod) bool {
			return !x.Enabled
		}))...)
		if l, ok := game.LoaderOf(gameID, s.profiles.LoaderID(gameID, id)); ok {
			if dir, err := s.profiles.ProfileDir(gameID, id); err == nil {
				r.LoadFailures = loaderFailures(l, loader.ProfileView{Game: gameID, Dir: dir}, s.playerLog(gameID, l), func() map[string]framework.Mod { return pluginOwners(mods) })
			}
			if f, bad := gameVersionFailure(gameID, l, env); bad {
				r.LoadFailures = append(r.LoadFailures, f)
			}
		}
		r.PluginClashes = pluginClashes(pkgs)
		if src, ok := thunderstoreSource(); ok {
			r.Deprecated = deprecatedPackages(ctx, src, thunderstoreKey(gameID), "", pkgs)
		}
		if s.Runs != nil && runID != "" {
			_, summary, err := s.Runs.LastRunSummary(gameID, id)
			if err == nil {
				r.RunErrors = RunErrorsFromSummary(runID, summary, mods)
			}
		} else if r.RunErrors == nil {
			r.RunErrors = []RunError{}
		}
		if s.meta != nil {
			enabledOnly := false
			if s.settings != nil {
				enabledOnly = s.settings.Get().CheckOnlyEnabledMods
			}
			updateStart := time.Now()
			ur := checkUpdates(ctx, s.metaFor(gameID), env, mods, enabledOnly, false, s.NexusFiles)
			asked := 0
			for _, x := range mods {
				if (profile.Source{Kind: x.SourceKind}).Bundled() {
					continue
				}
				if enabledOnly && !x.Enabled {
					continue
				}
				asked++
			}
			r.Timings = append(r.Timings, CheckTiming{Name: "updates", Ms: time.Since(updateStart).Milliseconds(), Count: asked})
			if s.updates != nil && !ur.Unknown {
				s.mu.Lock()
				s.updates[key] = cachedUpdates{fingerprint: fingerprint(env, mods, ""), at: time.Now(), result: ur}
				s.mu.Unlock()
			}
		}
		logCheckTimings(id, r.Timings)
		s.recordHealth(gameID, id, env, mods, r)
		return r
	})
	if err != nil {
		return Result{}, err
	}
	entry := cached{fingerprint: fp, result: r, sameJob: s.sameJobRows(gameID, id, mods)}
	s.mu.Lock()
	if r.Unknown {
		entry.until = time.Now().Add(unknownResultTTL)
	}
	s.cache[key] = entry
	s.mu.Unlock()
	return s.withDrift(gameID, id, fp, s.withDismissed(gameID, id, s.withCompat(ctx, r, gameID, mods, entry.sameJob)))
}

// ForgetCached drops every result and scan Mortar holds in memory, after the cache folder is cleared,
// so the next check reads and fetches everything afresh.
//
//wails:ignore
func (s *Service) ForgetCached() {
	s.mu.Lock()
	s.cache = map[string]cached{}
	s.drift = map[string]driftScan{}
	s.updates = map[string]cachedUpdates{}
	s.assets = map[string]cachedIndex{}
	s.mu.Unlock()
	framework.Forget()
}

// driftRescan is how long one drift scan answers repeated checks of an unchanged profile, such as the focus events
// of switching windows; a change to the mods or to the recorded baseline rescans at once.
const driftRescan = 5 * time.Second

type driftScan struct {
	key   string
	at    time.Time
	drift []profile.Drift
}

func (s *Service) withDrift(gameID, id, fp string, r Result) (Result, error) {
	r.Damaged = s.damagedMods(gameID, id)
	if s.settings != nil && !s.settings.Get().DriftChecksOn() {
		r.Drift = []profile.Drift{}
		return r, nil
	}
	memo := gameID + "/" + id
	key := fp + "|" + s.profiles.DriftBaseline(gameID, id)
	s.mu.Lock()
	last, ok := s.drift[memo]
	s.mu.Unlock()
	if ok && last.key == key && time.Since(last.at) < driftRescan {
		r.Drift = last.drift
		return r, nil
	}
	drift, err := s.profiles.ScanModsDrift(gameID, id)
	if err != nil {
		return Result{}, err
	}
	if drift == nil {
		drift = []profile.Drift{}
	}
	s.mu.Lock()
	s.drift[memo] = driftScan{key: key, at: time.Now(), drift: drift}
	s.mu.Unlock()
	r.Drift = drift
	return r, nil
}

func (s *Service) withDismissed(gameID, id string, r Result) Result {
	tokens := s.settings.Get().Dismissed[dismissBucket(gameID, id)]
	var dismissed []DismissedProblem
	r.AssetConflicts, dismissed = hideDismissed(r.AssetConflicts, tokens)
	r.Dismissed = append(r.Dismissed, dismissed...)
	r.Broken, dismissed = hideDismissedBroken(r.Broken, tokens)
	r.Dismissed = append(r.Dismissed, dismissed...)
	r.Missing, dismissed = hideDismissedListed(r.Missing, tokens)
	r.Dismissed = append(r.Dismissed, dismissed...)
	r.Settings, dismissed = hideDismissedSettings(r.Settings, tokens)
	r.Dismissed = append(r.Dismissed, dismissed...)
	return r
}

// DismissAbandonedMod hides an author-marked broken row for this profile until the mod is gone.
func (s *Service) DismissAbandonedMod(_ context.Context, gameID, id string, uniqueID mod.ID) error {
	if uniqueID == "" {
		return errors.New("missing mod id")
	}
	token := dismissToken("broken", uniqueID.Fold())
	return s.appendDismissed(dismissBucket(gameID, id), token)
}

// DismissListedRequirement hides a Nexus-listed requirement for this profile until it is gone.
func (s *Service) DismissListedRequirement(_ context.Context, gameID, id string, uniqueID mod.ID) error {
	if uniqueID == "" {
		return errors.New("missing requirement id")
	}
	token := dismissToken("listed", uniqueID.Fold())
	return s.appendDismissed(dismissBucket(gameID, id), token)
}

// DismissSetting hides one compatibility setting for this profile until its patch group is gone.
func (s *Service) DismissSetting(_ context.Context, gameID, id string, uniqueID mod.ID, field string) error {
	field = strings.TrimSpace(field)
	if uniqueID == "" || field == "" {
		return errors.New("missing setting")
	}
	token := dismissToken("setting", uniqueID.Fold()+"\t"+strings.ToLower(field))
	return s.appendDismissed(dismissBucket(gameID, id), token)
}

// RememberSettingChoice keeps a setting hint hidden while its chosen value remains current.
func (s *Service) RememberSettingChoice(_ context.Context, gameID, id string, uniqueID mod.ID, field, value string) error {
	field = strings.TrimSpace(field)
	if uniqueID == "" || field == "" {
		return errors.New("missing setting")
	}
	token := settingChoiceToken(uniqueID, field, value)
	return s.appendDismissed(dismissBucket(gameID, id), token)
}

// ConflictImageCrop returns a PNG data URL of uniqueID's FromFile cropped to x,y,w,h, clamped to the image.
func (s *Service) ConflictImageCrop(_ context.Context, gameID, id string, uniqueID mod.ID, fromFile string, x, y, w, h int) (string, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return "", err
	}
	fromFile = contentpatcher.ContentReference("", fromFile)
	if uniqueID == "" || fromFile == "" {
		return "", errors.New("missing pack or image")
	}
	for _, m := range mods {
		if !mod.Equal(m.ModID(), uniqueID) {
			continue
		}
		return contentpatcher.CropPackImage(m.Folder, fromFile, x, y, w, h)
	}
	return "", errors.New("unknown pack")
}

// DismissAssetConflict hides a Content Patcher overlap for this profile until it is gone.
func (s *Service) DismissAssetConflict(_ context.Context, gameID, id, kind, target string) error {
	kind, target = strings.TrimSpace(kind), strings.TrimSpace(target)
	if kind == "" || target == "" {
		return errors.New("missing conflict kind or target")
	}
	token := dismissToken(kind, target)
	return s.appendDismissed(dismissBucket(gameID, id), token)
}

func (s *Service) appendDismissed(bucket, token string) error {
	return s.settings.AppendDismissed(bucket, token)
}

func (s *Service) RestoreDismissed(_ context.Context, gameID, id, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("missing dismissal token")
	}
	bucket := dismissBucket(gameID, id)
	_, err := s.settings.Update(func(v *settings.Settings) {
		current := v.Dismissed[bucket]
		if !slices.Contains(current, token) {
			return
		}
		next := maps.Clone(v.Dismissed)
		next[bucket] = slices.Delete(slices.Clone(current), slices.Index(current, token), slices.Index(current, token)+1)
		v.Dismissed = next
	})
	return err
}

// Updates lists the newer versions SMAPI's API suggests for the profile's mods. The answer is kept for an hour
// or until the mods or versions change, unless SMAPI's API could not be reached, in which case the next call
// tries again.
func (s *Service) Updates(ctx context.Context, gameID, id string) (UpdatesResult, error) {
	return s.updatesFor(ctx, gameID, id, false)
}

// CheckUpdatesNow asks SMAPI's API again for every mod in the profile, ignoring cached answers: the explicit
// "Check for updates" (F5).
func (s *Service) CheckUpdatesNow(ctx context.Context, gameID, id string) (UpdatesResult, error) {
	return s.updatesFor(ctx, gameID, id, true)
}

func (s *Service) updatesFor(ctx context.Context, gameID, id string, fresh bool) (UpdatesResult, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return UpdatesResult{}, err
	}
	env := s.Environment(gameID)
	fp := fingerprint(env, mods, "")
	key := gameID + "/" + id
	s.mu.Lock()
	c, ok := s.updates[key]
	s.mu.Unlock()
	if ok && !fresh && c.fingerprint == fp && time.Since(c.at) < updatesTTL {
		return hideUpdates(c.result, mods, s.settings.Get()), nil
	}
	set := s.settings.Get()
	r := checkUpdates(ctx, s.metaFor(gameID), env, mods, set.CheckOnlyEnabledMods, fresh, s.NexusFiles)
	s.fixStaleManifests(gameID, id, r.Held)
	r.Updates = append(r.Updates, s.sourceUpdates(ctx, gameID, mods, r.Updates)...)
	if !r.Unknown {
		s.mu.Lock()
		s.updates[key] = cachedUpdates{fp, time.Now(), r}
		s.mu.Unlock()
	}
	return hideUpdates(r, mods, set), nil
}

// fixStaleManifests sets each manifest whose download is already the suggested version to that version, so SMAPI
// stops offering an update that is installed. Authors often forget to bump a manifest; the download's version is
// the truth.
func (s *Service) fixStaleManifests(gameID, id string, held []Held) {
	if s.profiles == nil {
		return
	}
	fixed := 0
	for _, h := range held {
		if h.Reason != HeldCurrent {
			continue
		}
		if err := s.profiles.FixStaleManifest(gameID, id, h.Key, h.ID, h.Version); err != nil {
			log.Printf("updates: %s: could not fix the manifest of %s: %v", id, h.ID, err)
			continue
		}
		fixed++
	}
	if fixed > 0 {
		log.Printf("updates: %s: set %d stale manifest versions to their download's version", id, fixed)
	}
}

func hideUpdates(r UpdatesResult, mods []framework.Mod, set settings.Settings) UpdatesResult {
	return HideHeld(r, mods, set.IncludePrereleaseModVersions, set.SmapiBuilds)
}

func (s *Service) recordHealth(gameID, id string, env Environment, mods []framework.Mod, r Result) {
	if s.profiles == nil {
		return
	}
	dir, err := s.profiles.ProfileDir(gameID, id)
	if err != nil {
		return
	}
	r = s.withDismissed(gameID, id, r)
	point := profile.HealthPoint{
		At:       time.Now().UTC(),
		Problems: r.Count(),
		Warnings: r.WarningCount(),
		Updates:  s.visibleUpdateCount(gameID, id, env, mods),
	}
	_ = profile.AppendHealth(dir, point)
}

func (s *Service) visibleUpdateCount(gameID, id string, env Environment, mods []framework.Mod) int {
	if s.settings == nil {
		return 0
	}
	fp := fingerprint(env, mods, "")
	key := gameID + "/" + id
	s.mu.Lock()
	c, ok := s.updates[key]
	s.mu.Unlock()
	if !ok || c.fingerprint != fp {
		return 0
	}
	return len(hideUpdates(c.result, mods, s.settings.Get()).Updates)
}

// UpdateWarning is the Play dialog after a game update: the last launched Stardew version versus the installed one.
type UpdateWarning struct {
	Changed   bool     `json:"changed"`
	Recorded  string   `json:"recorded"`
	Installed string   `json:"installed"`
	Broken    []Broken `json:"broken"`
}

// UpdateWarning reports whether the installed game version differs from the last successful launch,
// and which of this profile's mods the compatibility data marks broken for the installed version.
func (s *Service) UpdateWarning(ctx context.Context, gameID, id string) (UpdateWarning, error) {
	env := s.Environment(gameID)
	recorded := s.settings.Get().LastPlayed[gameID].GameVersion
	if !loader.VersionChanged(recorded, env.GameVersion) {
		return versionChangeWarning(recorded, env.GameVersion, nil), nil
	}
	mods, err := s.installed(gameID, id)
	if err != nil {
		return UpdateWarning{}, err
	}
	return versionChangeWarning(recorded, env.GameVersion, Check(ctx, s.metaFor(gameID), env, mods).Broken), nil
}

func versionChangeWarning(recorded, installed string, broken []Broken) UpdateWarning {
	out := UpdateWarning{Recorded: recorded, Installed: installed, Broken: []Broken{}}
	if !loader.VersionChanged(recorded, installed) {
		return out
	}
	out.Changed = true
	if broken == nil {
		return out
	}
	out.Broken = broken
	return out
}

// Relations says what the mod key/uniqueID needs, which mods need it and where its page is.
func (s *Service) Relations(gameID, id, key string, uniqueID mod.ID) (Relations, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return Relations{}, err
	}
	r, ok := Relate(versionScheme(gameID), mods, nexusDomain(gameID), key, uniqueID)
	if !ok {
		return Relations{}, errors.New("no such mod in this profile")
	}
	return r, nil
}

// Pages says where each mod's page is, keyed "key/id"; mods without a known page are left out.
func (s *Service) Pages(gameID, id string) (map[string]string, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return nil, err
	}
	return Pages(mods, nexusDomain(gameID)), nil
}

// maxDamagedFiles bounds the file names a Damaged row carries.
const maxDamagedFiles = 5

// damagedMods is the profile's mods whose store item failed verification, one row per item.
func (s *Service) damagedMods(gameID, id string) []Damaged {
	if s.Damage == nil {
		return nil
	}
	damaged := s.Damage(gameID)
	if len(damaged) == 0 {
		return nil
	}
	mods, err := s.installed(gameID, id)
	if err != nil {
		return nil
	}
	return damagedRows(mods, damaged)
}

func damagedRows(mods []framework.Mod, damaged map[string]store.Damage) []Damaged {
	var out []Damaged
	for _, m := range mods {
		d, ok := damaged[m.Key]
		if !ok || slices.ContainsFunc(out, func(x Damaged) bool { return x.Key == m.Key }) {
			continue
		}
		files := slices.Concat(d.Missing, d.Changed, d.Extra)
		out = append(out, Damaged{
			Key: m.Key, Name: m.Name, Missing: len(d.Missing), Changed: len(d.Changed), Extra: len(d.Extra),
			Files: files[:min(len(files), maxDamagedFiles)],
		})
	}
	return out
}
