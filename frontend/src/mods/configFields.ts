import type { ConfigNode, Hint } from './configForm.ts'

interface CPField {
  allowValues: string[]
  allowMultiple: boolean
  allowBlank: boolean
  defaultValue: string
  description: string
  section: string
}

type CPSchema = Record<string, CPField>

const wordSplit = /\s+/
const acronym = /^[A-Z]{2,}$/
const camel = /([a-z0-9])([A-Z])/g
const acronymBreak = /([A-Z]+)([A-Z][a-z])/g

function splitCSV(raw: string): string[] {
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

function asString(v: unknown): string {
  if (typeof v === 'string') {
    return v
  }
  if (typeof v === 'boolean' || typeof v === 'number') {
    return String(v)
  }
  return ''
}

function hintOf(field: CPField): Hint {
  return {
    description: field.description,
    defaultValue: field.defaultValue,
    section: field.section,
  }
}

function overlay(node: ConfigNode, field: CPField): ConfigNode {
  if (field.allowValues.length > 0 && (node.kind === 'string' || node.kind === 'keybind')) {
    return {
      kind: 'choice',
      value: node.value,
      options: field.allowValues,
      multiple: field.allowMultiple,
      allowBlank: field.allowBlank,
      defaultValue: field.defaultValue,
      description: field.description,
      section: field.section,
    }
  }
  if (
    node.kind === 'bool' ||
    node.kind === 'int' ||
    node.kind === 'float' ||
    node.kind === 'string' ||
    node.kind === 'keybind'
  ) {
    return { ...node, hint: hintOf(field) }
  }
  return node
}

function coerce(node: ConfigNode, raw: string): ConfigNode {
  switch (node.kind) {
    case 'bool':
      return { ...node, value: raw === 'true' }
    case 'int':
    case 'float':
      return { ...node, value: raw }
    case 'string':
    case 'keybind':
      return { ...node, value: raw }
    case 'choice':
      return { ...node, value: raw }
    default:
      return node
  }
}

function setParts(node: ConfigNode, parts: string[], raw: string): ConfigNode {
  if (parts.length === 0) {
    return coerce(node, raw)
  }
  const [head, ...rest] = parts
  if (node.kind !== 'object') {
    return node
  }
  return {
    kind: 'object',
    entries: node.entries.map((e) =>
      e.key === head ? { key: e.key, node: setParts(e.node, rest, raw) } : e,
    ),
  }
}

function fieldLabel(name: string): string {
  const parts = name
    .replace(/_/g, ' ')
    .replace(camel, '$1 $2')
    .replace(acronymBreak, '$1 $2')
    .trim()
    .split(wordSplit)
    .filter(Boolean)
  return parts
    .map((w) => (acronym.test(w) ? w : `${w.charAt(0).toUpperCase()}${w.slice(1)}`))
    .join(' ')
}

function parseCPSchema(content: unknown): CPSchema {
  if (!content || typeof content !== 'object') {
    return {}
  }
  const root = content as Record<string, unknown>
  const raw = (root.ConfigSchema ?? root) as Record<string, unknown>
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return {}
  }
  const out: CPSchema = {}
  for (const [key, spec] of Object.entries(raw)) {
    if (spec && typeof spec === 'object' && !Array.isArray(spec)) {
      const o = spec as Record<string, unknown>
      out[key] = {
        allowValues: typeof o.AllowValues === 'string' ? splitCSV(o.AllowValues) : [],
        allowMultiple: o.AllowMultiple === true,
        allowBlank: o.AllowBlank === true,
        defaultValue: asString(o.Default),
        description: typeof o.Description === 'string' ? o.Description : '',
        section: typeof o.Section === 'string' ? o.Section : '',
      }
    }
  }
  return out
}

function applyCPSchema(node: ConfigNode, schema: CPSchema): ConfigNode {
  if (node.kind !== 'object') {
    return node
  }
  return {
    kind: 'object',
    entries: node.entries.map((e) => {
      const field = schema[e.key]
      const child = e.node.kind === 'object' ? applyCPSchema(e.node, schema) : e.node
      return { key: e.key, node: field ? overlay(child, field) : child }
    }),
  }
}

function setByPath(node: ConfigNode, path: string, raw: string): ConfigNode {
  return setParts(node, path.split('.').filter(Boolean), raw)
}

export { applyCPSchema, fieldLabel, parseCPSchema, setByPath }
