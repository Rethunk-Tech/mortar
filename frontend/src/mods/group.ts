import type { Entry } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'

const MAX_ENTRY_NOTE = 500
const MAX_ENTRY_TAGS = 8
const MAX_ENTRY_TAG = 24

const GROUP_BY_IDS = ['none', 'category', 'source', 'tag'] as const

type GroupBy = (typeof GROUP_BY_IDS)[number]

function emptyGroupLabel(
  by: GroupBy,
  labels: { category: string; source: string; tag: string },
): string {
  if (by === 'category') {
    return labels.category
  }
  if (by === 'source') {
    return labels.source
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
  out.sort((a, b) => a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' }))
  return out
}

function sanitizeListGroupBy(by: string | null | undefined): GroupBy {
  if (by === 'category' || by === 'source' || by === 'tag' || by === 'none') {
    return by
  }
  return 'none'
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
  const named = keys
    .filter((k) => k !== '')
    .sort((a, b) => a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' }))
  const order = keys.includes('') ? [...named, ''] : named
  return order.map((key) => ({ key, items: sortedByKey(buckets.get(key) ?? []) }))
}

export type { Group, GroupBy }
export {
  emptyGroupLabel,
  firstTag,
  GROUP_BY_IDS,
  groupSorted,
  MAX_ENTRY_NOTE,
  MAX_ENTRY_TAG,
  MAX_ENTRY_TAGS,
  profileTags,
  sanitizeListGroupBy,
  takeTags,
}
