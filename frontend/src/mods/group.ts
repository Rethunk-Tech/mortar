import type {
  CustomCategory,
  Entry,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { readStored, writeStored } from '../shell/useStoredState.ts'
import { cmpText } from './cmpText.ts'
import { idKey } from './dependents.ts'

const MAX_ENTRY_NOTE = 500
const MAX_ENTRY_TAGS = 8
const MAX_ENTRY_TAG = 24

const GROUP_BY_IDS = [
  'none',
  'status',
  'category',
  'source',
  'tag',
  'framework',
  'author',
  'group',
] as const

const STATUS_GROUP_ORDER = ['problems', 'update', 'enabled', 'disabled'] as const

const SMAPI_MODS_GROUP = 'smapi'

type GroupBy = (typeof GROUP_BY_IDS)[number]
type StatusGroup = (typeof STATUS_GROUP_ORDER)[number]

function emptyGroupLabel(
  by: GroupBy,
  labels: { category: string; source: string; tag: string; author: string; group: string },
): string {
  if (by === 'category') {
    return labels.category
  }
  if (by === 'source') {
    return labels.source
  }
  if (by === 'author') {
    return labels.author
  }
  if (by === 'group') {
    return labels.group
  }
  return labels.tag
}

function firstTag(tags: readonly string[] | null | undefined): string {
  if (!tags) {
    return ''
  }
  for (const raw of tags) {
    const tag = raw.trim()
    if (tag !== '') {
      return tag
    }
  }
  return ''
}

function takeTags(next: readonly string[]): string[] {
  const cleaned: string[] = []
  const seen = new Set<string>()
  for (const raw of next) {
    const tag = raw.trim()
    if (tag !== '' && tag.length <= MAX_ENTRY_TAG && !seen.has(tag.toLowerCase())) {
      seen.add(tag.toLowerCase())
      cleaned.push(tag)
      if (cleaned.length === MAX_ENTRY_TAGS) {
        break
      }
    }
  }
  return cleaned
}

function profileTags(entries: readonly Entry[] | null | undefined): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const entry of entries ?? []) {
    for (const raw of entry.tags ?? []) {
      const tag = raw.trim()
      if (tag !== '') {
        const key = tag.toLowerCase()
        if (!seen.has(key)) {
          seen.add(key)
          out.push(tag)
        }
      }
    }
  }
  out.sort((a, b) => cmpText(a, b))
  return out
}

function isGroupBy(by: string | null | undefined): by is GroupBy {
  return (GROUP_BY_IDS as readonly (string | null | undefined)[]).includes(by)
}

function sanitizeListGroupBy(by: string | null | undefined): GroupBy {
  return isGroupBy(by) ? by : 'status'
}

function statusGroupKey(hasProblem: boolean, hasUpdate: boolean, enabled: boolean): StatusGroup {
  if (hasProblem) {
    return 'problems'
  }
  if (hasUpdate) {
    return 'update'
  }
  if (enabled) {
    return 'enabled'
  }
  return 'disabled'
}

function frameworkGroupKey(
  contentPackFor: string | null | undefined,
  uniqueId: string,
  names: ReadonlyMap<string, string>,
): string {
  const packFor = contentPackFor?.trim() ?? ''
  if (packFor === '' || idKey(packFor) === idKey(uniqueId)) {
    return SMAPI_MODS_GROUP
  }
  const name = names.get(idKey(packFor))?.trim() ?? ''
  return name === '' ? packFor : name
}

interface Group<T> {
  key: string
  items: T[]
}

function groupSorted<T>(
  items: readonly T[],
  by: GroupBy,
  keyOf: (item: T) => string,
  compare: (a: T, b: T) => number,
): Group<T>[] {
  const sortedByKey = (list: T[]) => [...list].sort(compare)
  if (by === 'none') {
    return [{ key: '', items: sortedByKey([...items]) }]
  }
  const buckets = new Map<string, T[]>()
  for (const item of items) {
    const key = keyOf(item)
    const list = buckets.get(key)
    if (list) {
      list.push(item)
    } else {
      buckets.set(key, [item])
    }
  }
  const keys = [...buckets.keys()]
  const order = orderedGroupKeys(by, keys)
  return order.map((key) => ({ key, items: sortedByKey(buckets.get(key) ?? []) }))
}

const COLLAPSED_PREFIX = 'mortar.modsCollapsed.'

const isRecord = (v: unknown): v is Record<string, unknown> =>
  typeof v === 'object' && v !== null && !Array.isArray(v)

function loadCollapsed(game: string): Record<string, boolean> {
  if (game === '') {
    return {}
  }
  const parsed = readStored<Record<string, unknown>>(COLLAPSED_PREFIX + game, {}, isRecord)
  const out: Record<string, boolean> = {}
  for (const [key, value] of Object.entries(parsed)) {
    if (value === true) {
      out[key] = true
    }
  }
  return out
}

function persistCollapsed(game: string, collapsed: Record<string, boolean>) {
  if (game === '') {
    return
  }
  const stored: Record<string, true> = {}
  for (const key of Object.keys(collapsed).filter((k) => collapsed[k])) {
    stored[key] = true
  }
  writeStored(COLLAPSED_PREFIX + game, stored)
}

function toggleCollapsed(
  game: string,
  cur: Record<string, boolean>,
  key: string,
  open: boolean,
): Record<string, boolean> {
  const next = { ...cur, [key]: open }
  persistCollapsed(game, next)
  return next
}

function installedNames(mods: readonly { uniqueId: string; name: string }[]): Map<string, string> {
  return new Map(mods.map((m) => [idKey(m.uniqueId), m.name] as const))
}

function customCategoryById(
  categories: readonly CustomCategory[],
): ReadonlyMap<string, CustomCategory> {
  return new Map(categories.map((c) => [c.id, c] as const))
}

function resolvedCategoryLabel(
  categoryOverride: string | undefined,
  nexusCategory: string | undefined,
  customById: ReadonlyMap<string, CustomCategory>,
): string {
  const override = categoryOverride?.trim() ?? ''
  if (override !== '') {
    const custom = customById.get(override)
    if (custom) {
      return custom.name
    }
    return override
  }
  return nexusCategory?.trim() ?? ''
}

interface GroupRow {
  source: string
  tags: readonly string[]
  categoryOverride?: string
  details?: { category?: string }
  groupName?: string
  mod: {
    uniqueId: string
    author: string
    enabled: boolean
    needs?: string[] | null
    optional?: string[] | null
    contentPackFor?: string | null
  }
}

function rowGroupKey(
  by: GroupBy,
  row: GroupRow,
  ctx: {
    hasProblem: boolean
    hasUpdate: boolean
    names: ReadonlyMap<string, string>
    customById: ReadonlyMap<string, CustomCategory>
  },
): string {
  if (by === 'category') {
    return resolvedCategoryLabel(row.categoryOverride, row.details?.category, ctx.customById)
  }
  if (by === 'source') {
    return row.source
  }
  if (by === 'tag') {
    return firstTag(row.tags)
  }
  if (by === 'author') {
    return row.mod.author.trim()
  }
  if (by === 'status') {
    return statusGroupKey(ctx.hasProblem, ctx.hasUpdate, row.mod.enabled)
  }
  if (by === 'framework') {
    return frameworkGroupKey(row.mod.contentPackFor, row.mod.uniqueId, ctx.names)
  }
  if (by === 'group') {
    return row.groupName ?? ''
  }
  return ''
}

function groupHeading(
  by: GroupBy,
  key: string,
  labels: {
    empty: string
    problems: string
    update: string
    enabled: string
    disabled: string
    smapi: string
  },
): string {
  if (by === 'status') {
    if (key === 'problems') {
      return labels.problems
    }
    if (key === 'update') {
      return labels.update
    }
    if (key === 'enabled') {
      return labels.enabled
    }
    return labels.disabled
  }
  if (by === 'framework' && key === SMAPI_MODS_GROUP) {
    return labels.smapi
  }
  return key || labels.empty
}
type HeadingCopy = {
  category: string
  source: string
  tag: string
  author: string
  group: string
  problems: string
  update: string
  enabled: string
  disabled: string
  smapi: string
}

function listHeadingFor(groupBy: GroupBy, copy: HeadingCopy) {
  return (key: string) =>
    groupHeading(groupBy, key, {
      empty: emptyGroupLabel(groupBy, copy),
      problems: copy.problems,
      update: copy.update,
      enabled: copy.enabled,
      disabled: copy.disabled,
      smapi: copy.smapi,
    })
}


function orderedGroupKeys(by: GroupBy, keys: readonly string[]): string[] {
  if (by === 'status') {
    return STATUS_GROUP_ORDER.filter((k) => keys.includes(k))
  }
  const named = keys.filter((k) => k !== '' && k !== SMAPI_MODS_GROUP).sort((a, b) => cmpText(a, b))
  const tail: string[] = []
  if (keys.includes(SMAPI_MODS_GROUP)) {
    tail.push(SMAPI_MODS_GROUP)
  listHeadingFor,
  }
  if (keys.includes('')) {
    tail.push('')
  }
  return [...named, ...tail]
}

export type { GroupBy }
export {
  customCategoryById,
  emptyGroupLabel,
  firstTag,
  frameworkGroupKey,
  groupHeading,
  groupSorted,
  installedNames,
  loadCollapsed,
  MAX_ENTRY_NOTE,
  MAX_ENTRY_TAG,
  persistCollapsed,
  profileTags,
  resolvedCategoryLabel,
  rowGroupKey,
  SMAPI_MODS_GROUP,
  sanitizeListGroupBy,
  statusGroupKey,
  takeTags,
  toggleCollapsed,
}
