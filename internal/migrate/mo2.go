package migrate

import (
	"cmp"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
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

// mo2Installations finds the MO2 instances managing the game whose Nexus domain is domain.
func mo2Installations(home, modsPath, domain string) ([]installation, error) {
	info, _ := components.BundledGameByNexusDomain(domain)
	gameName := info.Name
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
		inst, ok, instErr := mo2Installation(filepath.Join(root, entry.Name()), modsPath, gameName)
		if instErr != nil {
			return nil, instErr
		}
		if !ok {
			continue
		}
		out = append(out, inst)
	}
	portable, ok, err := mo2Installation(home, modsPath, gameName)
	if err != nil {
		return nil, err
	}
	if ok {
		out = append(out, portable)
	}
	return out, nil
}

func mo2LocalAppData(home string) string {
	if runtime.GOOS == "windows" {
		actual, _ := os.UserHomeDir()
		if filepath.Clean(home) == filepath.Clean(actual) {
			if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
				return dir
			}
		}
		if dir := os.Getenv("LOCALAPPDATA"); dir != "" && home == "" {
			return dir
		}
		return filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(home, "AppData", "Local")
}

func mo2Installation(root, fallbackModsPath, gameName string) (installation, bool, error) {
	profiles, modsPath, err := mo2Profiles(root, fallbackModsPath, gameName)
	if err != nil {
		return installation{}, false, err
	}
	if len(profiles) == 0 {
		return installation{}, false, nil
	}
	name := filepath.Base(root)
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "Mod Organizer 2"
	}
	return installation{
		info:     SourceInfo{Kind: KindMO2, Name: name, Profiles: profileInfos(profiles)},
		root:     root,
		modsPath: modsPath,
	}, true, nil
}

func mo2Profiles(root, fallbackModsPath, gameName string) ([]ProfilePreview, string, error) {
	if !mo2IsGameInstance(root, gameName) {
		return nil, "", nil
	}
	modsPath := filepath.Join(root, "mods")
	if fallbackModsPath != "" {
		if _, err := os.Stat(modsPath); err != nil {
			modsPath = filepath.Clean(fallbackModsPath)
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, "profiles"))
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
		preview, previewErr := mo2Preview(root, modsPath, entry.Name())
		if previewErr != nil {
			return nil, "", previewErr
		}
		out = append(out, preview)
	}
	return out, modsPath, nil
}

func mo2IsGameInstance(root, gameName string) bool {
	data, err := fsx.ReadFile(filepath.Join(root, "ModOrganizer.ini"))
	if err != nil {
		return false
	}
	return gameName != "" && strings.EqualFold(iniGeneral(data)[mo2INIGameKey], gameName)
}

func mo2Preview(root, modsPath, id string) (ProfilePreview, error) {
	listPath := filepath.Join(root, "profiles", id, "modlist.txt")
	data, err := fsx.ReadFile(listPath)
	if err != nil {
		return ProfilePreview{}, err
	}
	listed := parseMO2Modlist(data)
	staged, err := folderMods(modsPath)
	if err != nil {
		return ProfilePreview{}, err
	}
	byBase := map[string][]folderMod{}
	for _, mod := range staged {
		byBase[filepath.Base(mod.Path)] = append(byBase[filepath.Base(mod.Path)], mod)
	}
	var out []ModPreview
	var missing []string
	for _, entry := range listed {
		out, missing = appendMO2Mod(out, missing, modsPath, byBase, entry)
	}
	if out == nil {
		out = []ModPreview{}
	}
	return ProfilePreview{
		ID: id, Name: id, Source: KindMO2, ModsPath: modsPath, Mods: out, Missing: missing,
	}, nil
}

func appendMO2Mod(out []ModPreview, missing []string, modsPath string, byBase map[string][]folderMod, entry mo2Listed) ([]ModPreview, []string) {
	modDir := filepath.Join(modsPath, entry.name)
	meta := readMO2Meta(modDir)
	items := byBase[entry.name]
	if entry.enabled && len(items) == 0 && meta.modID == 0 && !manifest.LoaderManaged(mod.SMAPI(entry.name)) {
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
	data, err := fsx.ReadFile(filepath.Join(modDir, "meta.ini"))
	if err != nil {
		return mo2Meta{}
	}
	general := iniGeneral(data)
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

func iniGeneral(data []byte) map[string]string {
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
		if !ok || section != "general" {
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
