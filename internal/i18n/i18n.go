// Package i18n translates the English text Go itself puts in front of the user through the OS: the tray menu, desktop
// notifications, native dialogs and the dialogs shown before the window opens. Everything else is translated in the
// frontend. Catalogs are gettext PO files with the English text as msgid and {name} placeholders, edited in Hosted
// Weblate like the frontend's; a message with no translation reads in English.
package i18n

import (
	"embed"
	"log"
	"maps"
	"path"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

//go:embed locales/*/native.po
var files embed.FS

type catalog map[string]string

var catalogs = sync.OnceValue(func() map[string]catalog {
	dirs, err := files.ReadDir("locales")
	if err != nil {
		log.Printf("i18n: %v", err)
		return nil
	}
	out := make(map[string]catalog, len(dirs))
	for _, d := range dirs {
		f, err := files.Open(path.Join("locales", d.Name(), "native.po"))
		if err != nil {
			log.Printf("i18n: %v", err)
			continue
		}
		c, err := parsePO(f)
		_ = f.Close()
		if err != nil {
			log.Printf("i18n: %s: %v", d.Name(), err)
			continue
		}
		out[d.Name()] = c
	}
	return out
})

// Locales are the locales Mortar ships, one directory each under locales/ and English first. The language setting
// accepts exactly these (and the empty string, which follows the system), and a test keeps them equal to the frontend's
// availableLocales.
var Locales = sync.OnceValue(func() []string {
	dirs, err := files.ReadDir("locales")
	if err != nil {
		log.Printf("i18n: %v", err)
		return []string{"en"}
	}
	out := []string{"en"}
	for _, d := range dirs {
		if d.Name() != "en" {
			out = append(out, d.Name())
		}
	}
	return out
})

// Valid reports whether language is a setting value: "" for the system language, or a shipped locale.
func Valid(language string) bool { return language == "" || slices.Contains(Locales(), language) }

var languageSource atomic.Pointer[func() string]

// SetLanguageSource sets where the user's language setting is read from; it is read on every call, so a change takes
// effect at once. Without a source, or while it returns "", Mortar follows the system language.
func SetLanguageSource(f func() string) { languageSource.Store(&f) }

// Current is the locale of the catalog that T uses.
func Current() string {
	setting := ""
	if f := languageSource.Load(); f != nil {
		setting = (*f)()
	}
	return resolve(catalogs(), setting, systemLanguage())
}

// T returns msgid in the current locale, with each {key} replaced by the value after it in kv. The msgid must be a
// string literal: a test checks every one against locales/en/native.po.
func T(msgid string, kv ...string) string {
	cats := catalogs()
	return format(translation(cats, Current(), msgid), msgid, kv)
}

// translation is msgid in locale, else in its base language (es-AR falls back to es), else "" for the English msgid.
func translation(cats map[string]catalog, locale, msgid string) string {
	if t := cats[locale][msgid]; t != "" {
		return t
	}
	if base, _, regional := strings.Cut(locale, "-"); regional {
		return cats[base][msgid]
	}
	return ""
}

// resolve picks the setting when it names a catalog, else the closest catalog to the system language: the same code,
// else the longest locale that is a prefix of it (es-MX takes es), else a locale of the same base language (pt-PT takes
// pt-BR), else en. This is the frontend's rule (bestLocale in src/i18n/locales.ts).
func resolve(cats map[string]catalog, setting, system string) string {
	if _, ok := cats[setting]; ok {
		return setting
	}
	system = strings.ToLower(system)
	locales := slices.Sorted(maps.Keys(cats))
	if i := slices.IndexFunc(locales, func(l string) bool { return strings.ToLower(l) == system }); i >= 0 {
		return locales[i]
	}
	best := ""
	for _, locale := range locales {
		if strings.HasPrefix(system, strings.ToLower(locale)+"-") && len(locale) > len(best) {
			best = locale
		}
	}
	if best != "" {
		return best
	}
	base, _, _ := strings.Cut(system, "-")
	for _, locale := range locales {
		if b, _, _ := strings.Cut(strings.ToLower(locale), "-"); b == base {
			return locale
		}
	}
	return "en"
}

func format(translated, msgid string, kv []string) string {
	msg := msgid
	if translated != "" {
		msg = translated
	}
	if len(kv) < 2 {
		return msg
	}
	pairs := make([]string, 0, len(kv))
	for i := 0; i+1 < len(kv); i += 2 {
		pairs = append(pairs, "{"+kv[i]+"}", kv[i+1])
	}
	return strings.NewReplacer(pairs...).Replace(msg)
}

// posixTag turns a POSIX locale such as "de_DE.UTF-8@euro" into the tag "de-DE"; "C" and "POSIX" name no language.
func posixTag(v string) string {
	if i := strings.IndexAny(v, ".@"); i >= 0 {
		v = v[:i]
	}
	if v == "C" || v == "POSIX" {
		return ""
	}
	return strings.ReplaceAll(v, "_", "-")
}
