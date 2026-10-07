import type { ConfigEntry, ConfigFile, ConfigValue } from './types.ts'

// A range this small reads better as a slider than as a field to type into.
const SLIDER_MAX_SPAN = 100

const sameValue = (a: ConfigValue, b: ConfigValue): boolean =>
  Array.isArray(a) && Array.isArray(b)
    ? a.length === b.length && a.every((v, i) => v === b[i])
    : a === b

const isModified = (e: ConfigEntry): boolean => !sameValue(e.value, e.default)

const wantsSlider = (e: ConfigEntry): boolean =>
  (e.type === 'int' || e.type === 'float') &&
  e.min !== undefined &&
  e.max !== undefined &&
  e.max - e.min <= SLIDER_MAX_SPAN

// A search matches the entry's key or description, and a section heading keeps all its entries.
function filterFile(file: ConfigFile, query: string): ConfigFile {
  const q = query.trim().toLowerCase()
  if (q === '') {
    return file
  }
  const sections = file.sections
    .map((s) =>
      s.name.toLowerCase().includes(q)
        ? s
        : {
            ...s,
            entries: s.entries.filter(
              (e) =>
                e.key.toLowerCase().includes(q) ||
                (e.label ?? '').toLowerCase().includes(q) ||
                e.description.toLowerCase().includes(q),
            ),
          },
    )
    .filter((s) => s.entries.length > 0)
  return { ...file, sections }
}

// A typed number: empty or unparsable keeps the previous value, and the range clamps the rest.
function parseNumber(raw: string, e: ConfigEntry): number | null {
  const n = e.type === 'int' ? Number.parseInt(raw, 10) : Number.parseFloat(raw)
  if (Number.isNaN(n)) {
    return null
  }
  const lo = e.min ?? Number.NEGATIVE_INFINITY
  const hi = e.max ?? Number.POSITIVE_INFINITY
  return Math.min(Math.max(n, lo), hi)
}

const HEX_COLOR = /^#?[0-9a-fA-F]{6}([0-9a-fA-F]{2})?$/
const isHexColor = (s: string): boolean => HEX_COLOR.test(s)

const modifiedCount = (file: ConfigFile): number =>
  file.sections.reduce((n, s) => n + s.entries.filter(isModified).length, 0)

const WORD_GAPS = [/([a-z])([A-Z])/g, /([A-Z]+)([A-Z][a-z])/g, /([A-Za-z])(\d)/g, /(\d)([A-Za-z])/g]
const SEPARATORS = /[_\-.\s]+/g

// A setting's key as words: split at camelCase and letter/number boundaries ("SpawnFreqCoal0To2" reads "Spawn Freq
// Coal 0 To 2"), so a label only ever breaks between words.
function humanizeKey(key: string): string {
  let text = key.replace(SEPARATORS, ' ')
  for (const gap of WORD_GAPS) {
    text = text.replace(gap, '$1 $2')
  }
  return text.trim()
}

export { filterFile, humanizeKey, isHexColor, isModified, modifiedCount, parseNumber, wantsSlider }
