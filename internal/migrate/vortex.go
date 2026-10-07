package migrate

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

type vortexProfile struct {
	ID       string                      `json:"id"`
	GameID   string                      `json:"gameId"`
	Name     string                      `json:"name"`
	ModState map[string]vortexProfileMod `json:"modState"`
}

type vortexProfileMod struct {
	Enabled bool `json:"enabled"`
}

type vortexMod struct {
	ID               string                     `json:"id"`
	InstallationPath string                     `json:"installationPath"`
	Attributes       map[string]json.RawMessage `json:"attributes"`
}

func vortexProfiles(root, domain string) ([]ProfilePreview, string, error) {
	state, err := readVortexState(root)
	if err != nil {
		return nil, "", err
	}
	profiles := vortexProfileList(state, domain)
	if len(profiles) == 0 {
		return nil, "", nil
	}
	modsPath := vortexModsPath(root, domain, state)
	out := make([]ProfilePreview, 0, len(profiles))
	for _, profile := range profiles {
		preview, err := vortexPreviewState(modsPath, domain, state, profile.ID)
		if err != nil {
			return nil, "", err
		}
		out = append(out, preview)
	}
	return out, modsPath, nil
}

func vortexPreview(root, domain, id string) (ProfilePreview, error) {
	state, err := readVortexState(root)
	if err != nil {
		return ProfilePreview{}, err
	}
	modsPath := vortexModsPath(root, domain, state)
	return vortexPreviewState(modsPath, domain, state, id)
}

func vortexPreviewState(modsPath, domain string, state map[string]json.RawMessage, id string) (ProfilePreview, error) {
	profiles := vortexProfileList(state, domain)
	var selected *vortexProfile
	for i := range profiles {
		if profiles[i].ID == id {
			selected = &profiles[i]
			break
		}
	}
	if selected == nil {
		return ProfilePreview{}, errors.New("Vortex profile " + id + " not found")
	}
	mods := vortexModList(state, domain)
	byID := make(map[string]vortexMod, len(mods))
	for _, vm := range mods {
		byID[vm.ID] = vm
	}
	staged, err := folderMods(modsPath)
	if err != nil {
		return ProfilePreview{}, err
	}
	byPath := make(map[string][]folderMod)
	for _, vm := range staged {
		byPath[filepath.Clean(vm.Path)] = append(byPath[filepath.Clean(vm.Path)], vm)
	}
	modIDs := make(map[string]bool)
	for _, vm := range mods {
		modIDs[vm.ID] = true
	}
	for id := range selected.ModState {
		modIDs[id] = true
	}
	var out []ModPreview
	var missing []string
	for modID := range modIDs {
		vm := byID[modID]
		enabled := selected.ModState[modID].Enabled
		path := vortexModPath(modsPath, vm)
		items := byPath[filepath.Clean(path)]
		if enabled && len(items) == 0 && !manifest.LoaderManaged(mod.SMAPI(modID)) &&
			!manifest.LoaderManaged(mod.SMAPI(rawString(vm.Attributes, "uniqueId", "uniqueID"))) {
			missing = append(missing, modID)
			continue
		}
		for _, item := range items {
			out = append(out, ModPreview{
				ID: item.ModID(), Name: item.Name, Version: item.Version,
				Enabled: enabled, NexusModID: nexusID(item.UpdateKeys), SourcePath: item.Path,
			})
		}
		if len(items) > 0 {
			continue
		}
		name := rawString(vm.Attributes, "name", "modName")
		if name == "" {
			name = modID
		}
		uniqueID := rawString(vm.Attributes, "uniqueId", "uniqueID")
		out = append(out, ModPreview{
			ID: mod.SMAPI(uniqueID), Name: name, Version: rawString(vm.Attributes, "version", "modVersion"),
			Enabled: enabled, NexusModID: rawInt(vm.Attributes, "modId"),
		})
	}
	if out == nil {
		out = []ModPreview{}
	}
	return ProfilePreview{
		ID: selected.ID, Name: selected.Name, Source: KindVortex, ModsPath: modsPath, Mods: out, Missing: missing,
	}, nil
}

// Vortex persists its Redux state as a LevelDB database at <root>/state.v2.
// Each leaf is stored under a "###"-joined path (hive###key###...) with a
// JSON-encoded value (src/main/src/store/LevelPersist.ts, SEPARATOR; values are
// written via JSON.stringify in ReduxPersistorIPC.ts).
const vortexKeySeparator = "###"

// ErrVortexRunning means Vortex holds its database files open. On Windows its write-ahead log and manifest are
// opened without read sharing, so the live state cannot be read, and a copy without them would be stale.
var ErrVortexRunning = errors.New("cannot read the data of a running Vortex: close Vortex and try again")

// readVortexState reads a private copy of the database: Vortex normally holds
// the LOCK, and a copy guarantees the live files are never written.
func readVortexState(root string) (map[string]json.RawMessage, error) {
	src := filepath.Join(root, "state.v2")
	info, err := os.Stat(src)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("read Vortex state %s: not a LevelDB directory", src)
	}
	tmp, err := os.MkdirTemp("", "mortar-vortex-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	if err := copyLevelDB(src, tmp); err != nil {
		return nil, err
	}
	db, err := leveldb.OpenFile(tmp, &opt.Options{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("read Vortex state %s: %w", src, err)
	}
	defer func() { _ = db.Close() }()
	return vortexTree(db)
}

// copyLevelDB copies regular files except LOCK. Table files go first and CURRENT
// last so a compaction racing the copy cannot leave CURRENT pointing at a
// manifest newer than the tables copied.
func copyLevelDB(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if e.Type().IsRegular() && e.Name() != "LOCK" {
			names = append(names, e.Name())
		}
	}
	rank := func(n string) int {
		switch {
		case strings.HasSuffix(n, ".ldb"), strings.HasSuffix(n, ".sst"):
			return 0
		case n == "CURRENT":
			return 2
		}
		return 1
	}
	sort.SliceStable(names, func(i, j int) bool { return rank(names[i]) < rank(names[j]) })
	for _, name := range names {
		data, err := fsx.ReadFile(filepath.Join(src, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if fileLocked(err) {
			return ErrVortexRunning
		}
		if err != nil {
			return err
		}
		if err := fsx.WriteFile(filepath.Join(dst, name), data, 0o600); err != nil {
			return err
		}
	}
	return nil
}

// vortexTree rebuilds the nested state map from the flat "###" keys. Parents
// sort before their children, so a blob stored at an intermediate path is
// extended by deeper leaf keys.
func vortexTree(db *leveldb.DB) (map[string]json.RawMessage, error) {
	root := map[string]any{}
	it := db.NewIterator(nil, nil)
	defer it.Release()
	for it.Next() {
		var value any
		if json.Unmarshal(it.Value(), &value) != nil {
			continue
		}
		parts := strings.Split(string(it.Key()), vortexKeySeparator)
		node := root
		for _, part := range parts[:len(parts)-1] {
			child, ok := node[part].(map[string]any)
			if !ok {
				child = map[string]any{}
				node[part] = child
			}
			node = child
		}
		last := parts[len(parts)-1]
		if existing, ok := node[last].(map[string]any); ok {
			if incoming, ok := value.(map[string]any); ok {
				maps.Copy(existing, incoming)
				continue
			}
		}
		node[last] = value
	}
	if err := it.Error(); err != nil {
		return nil, err
	}
	out := make(map[string]json.RawMessage, len(root))
	for k, v := range root {
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		out[k] = data
	}
	return out, nil
}

// vortexGameMatches is true for every game when domain is empty.
func vortexGameMatches(gameID, domain string) bool {
	return domain == "" || strings.EqualFold(gameID, domain)
}

func vortexProfileList(state map[string]json.RawMessage, domain string) []vortexProfile {
	persistent := objectValue(state, "persistent")
	raw := persistent["profiles"]
	var direct map[string]json.RawMessage
	if json.Unmarshal(raw, &direct) != nil {
		return nil
	}
	var out []vortexProfile
	for key, value := range direct {
		var profile vortexProfile
		if json.Unmarshal(value, &profile) != nil {
			continue
		}
		if profile.ID == "" {
			profile.ID = key
		}
		if profile.GameID != "" {
			if vortexGameMatches(profile.GameID, domain) {
				if profile.Name == "" {
					profile.Name = profile.ID
				}
				out = append(out, profile)
			}
			continue
		}
		var nested map[string]json.RawMessage
		if json.Unmarshal(value, &nested) != nil {
			continue
		}
		for nestedID, nestedValue := range nested {
			if err := json.Unmarshal(nestedValue, &profile); err != nil {
				continue
			}
			if !vortexGameMatches(profile.GameID, domain) {
				continue
			}
			if profile.ID == "" {
				profile.ID = nestedID
			}
			if profile.Name == "" {
				profile.Name = profile.ID
			}
			out = append(out, profile)
		}
	}
	return out
}

func vortexModList(state map[string]json.RawMessage, domain string) []vortexMod {
	persistent := objectValue(state, "persistent")
	games := objectValue(persistent, "mods")
	raw := games[domain]
	var values map[string]json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return nil
	}
	out := make([]vortexMod, 0, len(values))
	for key, value := range values {
		var vm vortexMod
		if json.Unmarshal(value, &vm) != nil {
			continue
		}
		if vm.ID == "" {
			vm.ID = key
		}
		out = append(out, vm)
	}
	return out
}

// vortexInstallPattern is the staging folder Vortex uses when none is stored
// (mod_management/util/getInstallPath.ts getInstallPathPattern).
const vortexInstallPattern = "{USERDATA}/{GAME}/mods"

var vortexPlaceholder = regexp.MustCompile(`(?i)\{(userdata|username|game)\}`)

// vortexModsPath resolves settings.mods.installPath.<game> like Vortex's
// resolveInstallPath: {USERDATA}, {USERNAME} and {GAME} expand case-insensitively
// and a relative result is relative to the Vortex data dir (root).
func vortexModsPath(root, domain string, state map[string]json.RawMessage) string {
	pattern := rawString(objectValue(objectValue(objectValue(state, "settings"), "mods"), "installPath"), domain)
	if pattern == "" {
		pattern = vortexInstallPattern
	}
	expanded := vortexPlaceholder.ReplaceAllStringFunc(pattern, func(m string) string {
		switch strings.ToLower(m[1 : len(m)-1]) {
		case "userdata":
			return root
		case "game":
			return domain
		}
		return vortexUsername()
	})
	return cleanVortexPath(root, filepath.FromSlash(strings.ReplaceAll(expanded, `\`, "/")))
}

func vortexUsername() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	name := u.Username
	return name[strings.LastIndexAny(name, `\/`)+1:]
}

func vortexModPath(modsPath string, vm vortexMod) string {
	path := cmp.Or(vm.InstallationPath, vm.ID)
	return cleanVortexPath(modsPath, path)
}

func objectValue(values map[string]json.RawMessage, key string) map[string]json.RawMessage {
	raw, ok := values[key]
	if !ok {
		return map[string]json.RawMessage{}
	}
	var out map[string]json.RawMessage
	if json.Unmarshal(raw, &out) != nil {
		return map[string]json.RawMessage{}
	}
	return out
}

func rawString(values map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		raw, ok := values[key]
		if !ok {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) == nil {
			return value
		}
	}
	return ""
}

func rawInt(values map[string]json.RawMessage, key string) int {
	raw, ok := values[key]
	if !ok {
		return 0
	}
	var number int
	if json.Unmarshal(raw, &number) == nil {
		return number
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		if _, err := fmt.Sscan(text, &number); err != nil {
			return 0
		}
	}
	return number
}

func cleanVortexPath(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, path)
}

// vortexMultiUser reads the flag Vortex stores at user###multiUser in the per-user database
// (src/main/src/Application.ts, setupPersistence: SubPersistor "user", key ["multiUser"]).
func vortexMultiUser(root string) bool {
	state, err := readVortexState(root)
	if err != nil {
		return false
	}
	return string(objectValue(state, "user")["multiUser"]) == "true"
}
