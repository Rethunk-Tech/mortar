import type {
  Entry,
  Mod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { customCategoryById } from './group.ts'
import { resolvedCategoryLabel } from './group.ts'

/** Every text a query can match for one mod: identity, tags, note, source and category names. */
export function searchFields(
  m: Mod,
  entry: Entry | undefined,
  nexusById: Record<number, { details?: { category?: string } } | undefined>,
  customById: ReturnType<typeof customCategoryById>,
): string[] {
  const nexusId = entry?.source.kind === 'nexus' ? (entry.source.modId ?? 0) : 0
  return [
    m.name,
    m.author,
    m.id,
    ...(entry?.tags ?? []),
    entry?.note ?? '',
    ...categoryNames(entry, nexusById[nexusId]?.details?.category, customById),
  ]
}

/** Category names a search can match: the resolved label (custom name or Nexus category) and the raw override. */
export function categoryNames(
  entry: Entry | undefined,
  nexusCategory: string | undefined,
  customById: ReturnType<typeof customCategoryById>,
): string[] {
  return [
    resolvedCategoryLabel(entry?.categoryOverride, nexusCategory, customById),
    entry?.categoryOverride ?? '',
    nexusCategory ?? '',
  ]
}

/** True when the trimmed, lowercased query is a substring of any field; an empty query matches everything. */
export function matchesQuery(q: string, fields: readonly string[]): boolean {
  return q === '' || fields.some((value) => value.toLowerCase().includes(q))
}

/** Every selected tag must be on the entry (AND). */
export function hasAllTags(
  entryTags: readonly string[] | null | undefined,
  selected: readonly string[],
) {
  return selected.every((tag) => (entryTags ?? []).includes(tag))
}
