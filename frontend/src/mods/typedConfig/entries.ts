import { looksLikeKeybind } from './keybind.ts'
import type { ConfigEntry, ConfigFile, ConfigValue } from './types.ts'

// A range this small reads better as a slider than as a field to type into.
const SLIDER_MAX_SPAN = 100
const RGB_LENGTH = 6
const RGBA_LENGTH = 8

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

// A picked "#rrggbb" written the way the mod wrote its value: with or without "#", in its letter case, and with its
// alpha digits kept, since a mod parses only the form it wrote.
function formatLike(picked: string, current: string): string {
  const rgb = picked.replace('#', '')
  const digits = current.replace('#', '')
  const alpha = digits.length === RGBA_LENGTH ? digits.slice(RGB_LENGTH) : ''
  const lower = digits === digits.toLowerCase() && digits !== digits.toUpperCase()
  const body = `${rgb}${alpha}`
  return `${current.startsWith('#') ? '#' : ''}${lower ? body.toLowerCase() : body.toUpperCase()}`
}

// A string entry holding a colour: hex digits under a colour-named key, or a value written with "#".
const COLOUR_KEY = /colou?r/i
const looksLikeColour = (e: ConfigEntry): boolean => {
  const value = String(e.value)
  return (
    isHexColor(value) && (value.startsWith('#') || COLOUR_KEY.test(`${e.key} ${e.label ?? ''}`))
  )
}

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

// "Languages Available: en, es, ru", "Options: Low/Medium/High", "Valid values: a | b": the values a description lists
// for a free-text entry. Two or more short values, or none.
const LISTED =
  /(?:available|options|valid values|possible values|supported values|choices|one of)[^:\n]{0,24}:\s*([^\n]+)/i
const MAX_OPTION_LENGTH = 32
const TRAILING_PUNCTUATION = /[.;)]+\s*$/
const OPTION_SEPARATOR = /\s*[,/|]\s*/
const RUN_OF_SPACES = /\s{2,}/

function suggestedOptions(description: string): string[] {
  const listed = LISTED.exec(description)?.[1]
  if (!listed) {
    return []
  }
  const options = listed
    .replace(TRAILING_PUNCTUATION, '')
    .split(OPTION_SEPARATOR)
    .map((o) => o.trim())
    .filter((o) => o !== '' && o.length <= MAX_OPTION_LENGTH && !RUN_OF_SPACES.test(o))
  return options.length >= 2 ? [...new Set(options)] : []
}

// A string entry holding a comma list: numbers or vectors ("200, 250, 350", "0, 0.8") get a field each, other short
// items chips. A keybind, colour or entry whose description lists its values has its own editor.
const MAX_CHIP_LENGTH = 64
const NUMERIC_ITEM = /^[-+]?(\d+\.?\d*|\.\d+)$/
const COMMA_SPACE = /,\s/
type CommaListKind = 'numbers' | 'chips'

const isNumericItem = (s: string): boolean => NUMERIC_ITEM.test(s)

const splitItems = (value: ConfigValue): string[] =>
  String(value)
    .split(',')
    .map((s) => s.trim())
    .filter((s) => s !== '')

function commaListKind(e: ConfigEntry): CommaListKind | null {
  if (looksLikeKeybind(e) || looksLikeColour(e)) {
    return null
  }
  const items = [e.value, e.default].map(splitItems).find((i) => i.length >= 2)
  if (!items) {
    return null
  }
  if (items.every(isNumericItem)) {
    return 'numbers'
  }
  return suggestedOptions(e.description).length === 0 &&
    items.every((i) => i.length <= MAX_CHIP_LENGTH)
    ? 'chips'
    : null
}

// The separator the value uses, or the default's once the value has no comma left; ", " when a comma is followed by
// a space.
function listSeparator(value: ConfigValue, fallback: ConfigValue): string {
  const source = String(value).includes(',') ? String(value) : String(fallback)
  return COMMA_SPACE.test(source) ? ', ' : ','
}

export {
  commaListKind,
  filterFile,
  formatLike,
  humanizeKey,
  isHexColor,
  isModified,
  isNumericItem,
  listSeparator,
  looksLikeColour,
  modifiedCount,
  parseNumber,
  splitItems,
  suggestedOptions,
  wantsSlider,
}
