import type { ConfigEntry, ConfigSection, ConfigValue, EntryType } from './types.ts'

// The backend writes every value as text in the file's own syntax: true/false, a number, a member, a string, or
// compact JSON for a list.
interface WireEntry {
  key: string
  label?: string
  type: string
  value: string
  default?: string
  hasDefault: boolean
  description?: string
  min?: number
  max?: number
  values?: string[]
  flags?: boolean
  labels?: string[]
  pending?: boolean
  readOnly?: boolean
  note?: string
}

interface WireSection {
  name: string
  entries: WireEntry[] | null
}

const KNOWN: readonly EntryType[] = ['bool', 'int', 'float', 'enum', 'string', 'color', 'list']

function parseValue(type: EntryType, text: string): ConfigValue {
  switch (type) {
    case 'bool':
      return text.trim().toLowerCase() === 'true'
    case 'int':
    case 'float': {
      const n = Number(text)
      return Number.isFinite(n) ? n : 0
    }
    case 'list':
      try {
        const parsed: unknown = JSON.parse(text)
        return Array.isArray(parsed) ? parsed.map(String) : []
      } catch {
        return []
      }
    default:
      return text
  }
}

function formatValue(value: ConfigValue): string {
  return Array.isArray(value) ? JSON.stringify(value) : String(value)
}

function toEntry(w: WireEntry): ConfigEntry {
  // A flags enum takes several members at once, which a single select cannot show.
  const known = KNOWN.find((k) => k === w.type) ?? 'string'
  const type: EntryType = known === 'enum' && w.flags ? 'string' : known
  const value = parseValue(type, w.value)
  return {
    key: w.key,
    ...(w.label ? { label: w.label } : {}),
    type,
    value,
    // Without a recorded default there is nothing to reset to, so the value stands as its own default.
    default: w.hasDefault ? parseValue(type, w.default ?? '') : value,
    description: w.description ?? '',
    ...(w.min === undefined ? {} : { min: w.min }),
    ...(w.max === undefined ? {} : { max: w.max }),
    ...(w.values ? { options: w.values } : {}),
    ...(w.labels ? { optionLabels: w.labels } : {}),
    ...(w.pending ? { pending: true } : {}),
    ...(w.readOnly ? { readOnly: true } : {}),
    ...(w.note ? { note: w.note } : {}),
  }
}

function toSections(wire: WireSection[]): ConfigSection[] {
  return wire.map((s) => ({ name: s.name, entries: (s.entries ?? []).map(toEntry) }))
}

export type { WireSection }
export { formatValue, toSections }
