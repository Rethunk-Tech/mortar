package migrate

import (
	"cmp"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/go-desktopenv/userdirs"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
)

const (
	mo2MetaModID  = "modid"
	mo2MetaVer    = "version"
	mo2MetaURL    = "url"
	mo2INIGameKey = "gamename"
)

type mo2Listed struct {
	name    string
	enabled bool
}

type mo2Meta struct {
	modID   int
	version string
}

// mo2Installations finds the MO2 instances managing the game MO2 calls gameName, plus the one in the chosen
// folder (an instance root, or the folder holding ModOrganizer.exe, which for a portable instance is the same).
func mo2Installations(home, chosen, modsPath, gameName, gameID string) ([]installation, error) {
	if gameName == "" {
		return nil, nil
	}
	var out []installation
	root := filepath.Join(mo2LocalAppData(home), "ModOrganizer")
	entries, err := os.ReadDir(root)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		inst, ok, instErr := mo2Installation(filepath.Join(root, entry.Name()), mo2Name+" ("+entry.Name()+")", modsPath, gameName, gameID)
		if instErr != nil {
			return nil, instErr
		}
		if !ok {
			continue
		}
		out = append(out, inst)
	}
	portable, ok, err := mo2Installation(home, mo2Name, modsPath, gameName, gameID)
	if err != nil {
		return nil, err
	}
	if ok {
		out = append(out, portable)
	}
	if chosen != "" && !slices.ContainsFunc(out, func(found installation) bool { return safefs.SamePath(found.root, chosen) }) {
		inst, ok, err := mo2Installation(chosen, mo2Name+" ("+filepath.Base(chosen)+")", modsPath, gameName, gameID)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, inst)
		}
	}
	return out, nil
}

func mo2LocalAppData(home string) string {
	if runtime.GOOS == "windows" {
		return userdirs.LocalAppData(home)
	}
	return filepath.Join(home, "AppData", "Local")
}

const mo2Name = "Mod Organizer 2"

// mo2Installation reads the MO2 instance at root; name is what the import list shows as its source.
func mo2Installation(root, name, fallbackModsPath, gameName, gameID string) (installation, bool, error) {
	profiles, modsPath, err := mo2Profiles(root, fallbackModsPath, gameName, gameID)
	if err != nil {
		return installation{}, false, err
	}
	if len(profiles) == 0 {
		return installation{}, false, nil
	}
	return installation{
		info:     SourceInfo{Kind: KindMO2, Name: name, Profiles: profileInfos(profiles)},
		root:     root,
		modsPath: modsPath,
		gameID:   gameID,
	}, true, nil
}

func mo2Profiles(root, fallbackModsPath, gameName, gameID string) ([]ProfilePreview, string, error) {
	if !mo2IsGameInstance(root, gameName) {
		return nil, "", nil
	}
	modsPath, profilesPath := mo2Dirs(root)
	if fallbackModsPath != "" {
		if _, err := os.Stat(modsPath); err != nil {
			modsPath = filepath.Clean(fallbackModsPath)
		}
	}
	entries, err := os.ReadDir(profilesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil
		}
		return nil, "", err
	}
	out := make([]ProfilePreview, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		preview, previewErr := mo2Preview(root, modsPath, gameID, entry.Name())
		if previewErr != nil {
			return nil, "", previewErr
		}
		out = append(out, preview)
	}
	return out, modsPath, nil
}

func mo2IsGameInstance(root, gameName string) bool {
	data, err := safefs.ReadFile(filepath.Join(root, "ModOrganizer.ini"))
	if err != nil {
		return false
	}
	return gameName != "" && strings.EqualFold(iniSection(data, "general")[mo2INIGameKey], gameName)
}

func mo2Preview(root, modsPath, gameID, id string) (ProfilePreview, error) {
	_, profilesPath := mo2Dirs(root)
	listPath := filepath.Join(profilesPath, id, "modlist.txt")
	data, err := safefs.ReadFile(listPath)
	if err != nil {
		return ProfilePreview{}, err
	}
	listed := parseMO2Modlist(data)
	staged, err := stagedMods(gameID, modsPath)
	if err != nil {
		return ProfilePreview{}, err
	}
	byBase := map[string][]folderMod{}
	for _, mod := range staged {
		byBase[filepath.Base(mod.Path)] = append(byBase[filepath.Base(mod.Path)], mod)
	}
	var out []ModPreview
	var missing []string
	smapiGame := isSMAPIGame(gameID)
	for _, entry := range listed {
		out, missing = appendMO2Mod(out, missing, modsPath, byBase, entry, smapiGame)
	}
	if out == nil {
		out = []ModPreview{}
	}
	return ProfilePreview{
		ID: id, Name: id, Source: KindMO2, ModsPath: modsPath, Mods: out, Missing: missing,
	}, nil
}

func appendMO2Mod(out []ModPreview, missing []string, modsPath string, byBase map[string][]folderMod, entry mo2Listed, smapiGame bool) ([]ModPreview, []string) {
	modDir := filepath.Join(modsPath, entry.name)
	meta := readMO2Meta(modDir)
	items := byBase[entry.name]
	if entry.enabled && len(items) == 0 && meta.modID == 0 && (!smapiGame || !smapimanifest.LoaderManaged(mod.SMAPI(entry.name))) {
		if _, err := os.Stat(modDir); err != nil {
			return out, append(missing, entry.name)
		}
	}
	for _, item := range items {
		nexus := meta.modID
		if nexus == 0 {
			nexus = nexusID(item.UpdateKeys)
		}
		out = append(out, ModPreview{
			ID: item.ModID(), Name: item.Name, Version: cmp.Or(item.Version, meta.version),
			Enabled: entry.enabled, NexusModID: nexus, SourcePath: localMO2Path(meta.modID, item.Path),
		})
	}
	if len(items) > 0 {
		return out, missing
	}
	out = append(out, ModPreview{
		ID: "", Name: entry.name, Version: meta.version,
		Enabled: entry.enabled, NexusModID: meta.modID, SourcePath: localMO2Path(meta.modID, modDir),
	})
	return out, missing
}

func localMO2Path(modID int, path string) string {
	if modID != 0 {
		return ""
	}
	return path
}

func readMO2Meta(modDir string) mo2Meta {
	data, err := safefs.ReadFile(filepath.Join(modDir, "meta.ini"))
	if err != nil {
		return mo2Meta{}
	}
	general := iniSection(data, "general")
	id := parseModID(general[mo2MetaModID])
	if id == 0 {
		id = nexusIDFromURL(general[mo2MetaURL])
	}
	return mo2Meta{modID: id, version: general[mo2MetaVer]}
}

func parseMO2Modlist(data []byte) []mo2Listed {
	seen := map[string]int{}
	var out []mo2Listed
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasSuffix(line, "_separator") {
			continue
		}
		if len(line) < 2 {
			continue
		}
		flag, name := line[0], line[1:]
		if flag == '*' || (flag != '+' && flag != '-') {
			continue
		}
		item := mo2Listed{name: name, enabled: flag == '+'}
		if i, ok := seen[name]; ok {
			out[i] = item
			continue
		}
		seen[name] = len(out)
		out = append(out, item)
	}
	return out
}

// mo2Dirs resolves the instance's mods and profiles folders the way MO2's
// PathSettings does (ModOrganizer2/modorganizer src/settings.cpp: base(),
// getConfigurablePath(), resolve()): [Settings] base_directory defaults to the
// folder holding ModOrganizer.ini, mod_directory and profiles_directory default
// to %BASE_DIR%/mods and %BASE_DIR%/profiles, and %BASE_DIR% expands to the base.
func mo2Dirs(root string) (mods, profiles string) {
	var settings map[string]string
	if data, err := safefs.ReadFile(filepath.Join(root, "ModOrganizer.ini")); err == nil {
		settings = iniSection(data, "settings")
	}
	// MO2 leaves a relative path relative to its own working directory, which is
	// the instance folder for the portable instances Mortar reads.
	abs := func(p string) string {
		p = filepath.FromSlash(strings.ReplaceAll(p, `\`, "/"))
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		return filepath.Clean(p)
	}
	base := root
	if b := mo2INIPath(settings["base_directory"]); b != "" {
		base = abs(b)
	}
	resolve := func(key, def string) string {
		path := cmp.Or(mo2INIPath(settings[key]), "%BASE_DIR%/"+def)
		path = strings.ReplaceAll(path, "%BASE_DIR%", filepath.ToSlash(base))
		return abs(path)
	}
	return resolve("mod_directory", "mods"), resolve("profiles_directory", "profiles")
}

// mo2INIPath decodes a path value as QSettings wrote it: a QByteArray is
// "@ByteArray(...)", and a QString with special characters is double-quoted,
// both with backslash escapes (a Windows path's backslashes are doubled).
func mo2INIPath(raw string) string {
	if inner, ok := strings.CutPrefix(raw, "@ByteArray("); ok {
		raw = strings.TrimSuffix(inner, ")")
	}
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' || i+1 == len(raw) {
			b.WriteByte(raw[i])
			continue
		}
		i++
		switch raw[i] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		default:
			b.WriteByte(raw[i])
		}
	}
	return b.String()
}

// iniSection reads the key=value pairs of the named section (lowercase), keys lowercased.
func iniSection(data []byte, want string) map[string]string {
	out := map[string]string{}
	section := ""
	text := strings.TrimPrefix(string(data), "\ufeff")
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || section != want {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	return out
}

func parseModID(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	id, err := strconv.Atoi(text)
	if err != nil {
		return 0
	}
	return id
}
