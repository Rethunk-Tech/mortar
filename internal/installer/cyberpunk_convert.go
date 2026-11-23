package installer

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Converting an archive mod into a REDmod is Vortex's redmod.conversion.ts: the archives of a mod that is nothing else
// move below mods/<module>/archives with a generated info.json, so the game's REDmod tool loads and orders them.

// REDmodExtensions are the files a REDmod's archives folder takes.
var redmodExtensions = []string{".archive", ".xl"}

// REDmod resolves modules by folder name, and a dot in it breaks the lookup; the rest are Windows-reserved.
var unusableInModuleName = regexp.MustCompile(`[.<>:"/\\|?*]`)

// REDmodModuleName is the folder name a converted mod's REDmod gets from the mod's name.
func REDmodModuleName(modName string) (string, error) {
	name := unusableInModuleName.ReplaceAllString(modName, "_")
	name = strings.TrimSpace(strings.Join(strings.Fields(name), " "))
	if name == "" {
		return "", fmt.Errorf("cannot build a REDmod name out of %q", modName)
	}
	return name, nil
}

// UniqueREDmodModuleName is wanted, or wanted with the first number from 2 that no taken module has: two REDmods cannot
// share a folder, and Windows compares folder names without regard to case.
func UniqueREDmodModuleName(wanted string, taken []string) string {
	has := func(name string) bool {
		return slices.ContainsFunc(taken, func(t string) bool { return strings.EqualFold(t, name) })
	}
	name := wanted
	for n := 2; has(name); n++ {
		name = fmt.Sprintf("%s %d", wanted, n)
	}
	return name
}

// REDmodConversion is an archive mod laid out as a REDmod: its files, and the info.json to put beside them.
type REDmodConversion struct {
	Files []File
	// InfoRel is where info.json goes below the game folder, and Info what it holds.
	InfoRel, Info string
}

// ConvertToREDmod moves the files of files, an archive mod's layout, to mods/<module>/archives. Only a mod that is
// nothing but .archive and .xl files in archive/pc/mod, with an .archive among them, can be converted.
func ConvertToREDmod(files []File, module, version string) (REDmodConversion, error) {
	var elsewhere, notArchives []string
	var loads bool
	out := make([]File, 0, len(files))
	dir := path.Join(redmodBase, module)
	for _, f := range files {
		if !isUnder(f.Rel, archivePrefix) {
			elsewhere = append(elsewhere, f.Rel)
			continue
		}
		ext := lower(path.Ext(f.Rel))
		if !slices.Contains(redmodExtensions, ext) {
			notArchives = append(notArchives, f.Rel)
		}
		loads = loads || ext == ".archive"
		f.Rel = path.Join(dir, "archives", f.Rel[len(archivePrefix)+1:])
		out = append(out, f)
	}
	switch {
	case len(elsewhere) > 0:
		return REDmodConversion{}, fmt.Errorf("only mods that are purely archives can be converted, and this one also has %s", strings.Join(elsewhere, ", "))
	case len(notArchives) > 0:
		return REDmodConversion{}, fmt.Errorf("a REDmod's archives folder only takes %s files, and this mod also has %s", strings.Join(redmodExtensions, " and "), strings.Join(notArchives, ", "))
	case !loads:
		return REDmodConversion{}, fmt.Errorf("there are no .archive files here to convert")
	}
	info, err := json.MarshalIndent(map[string]string{"name": module, "version": version}, "", "  ")
	if err != nil {
		return REDmodConversion{}, err
	}
	return REDmodConversion{Files: out, InfoRel: path.Join(dir, redmodInfoFile), Info: string(info)}, nil
}
