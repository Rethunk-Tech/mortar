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
  categories: {
    customById: ReturnType<typeof customCategoryById>
    siteCategories?: readonly string[] | null | undefined
  },
): string[] {
  const nexusId = entry?.source.kind === 'nexus' ? (entry.source.modId ?? 0) : 0
  return [
    m.name,
    m.author,
    m.id,
    ...(entry?.tags ?? []),
    entry?.note ?? '',
    ...categoryNames(
      entry,
      siteCategory(entry, nexusById[nexusId]?.details?.category),
      categories.customById,
      categories.siteCategories,
    ),
  ]
}

/** The category the mod's site gives it: a Thunderstore package's, recorded at install, or the Nexus page's. */
export function siteCategory(
  entry: Entry | undefined,
  nexusCategory: string | undefined,
): string | undefined {
  return entry?.source.kind === 'thunderstore' ? entry.source.category : nexusCategory
}

/** Every site category an installed Thunderstore package has, in the site's order: those the cached listing gives, else
 * the one recorded when it was installed. Nothing for any other source. */
export function siteCategories(
  entry: Entry | undefined,
  listed: readonly string[] | null | undefined,
): string[] | undefined {
  if (entry?.source.kind !== 'thunderstore') {
    return undefined
  }
  if (listed && listed.length > 0) {
    return [...listed]
  }
  return entry.source.category ? [entry.source.category] : undefined
}

/** Category names a search can match: the resolved label (custom name or Nexus category), the raw override, and every
 * site category of a Thunderstore package. */
export function categoryNames(
  entry: Entry | undefined,
  nexusCategory: string | undefined,
  customById: ReturnType<typeof customCategoryById>,
  listed?: readonly string[] | null,
): string[] {
  return [
    resolvedCategoryLabel(entry?.categoryOverride, nexusCategory, customById),
    entry?.categoryOverride ?? '',
    nexusCategory ?? '',
    ...(listed ?? []),
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
