import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type {
  CustomCategory,
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { customCategoryById, resolvedCategoryLabel } from './group.ts'
import type { ListRow } from './listColumns.ts'
import { kindLabel, nexusIdOf, sourceKind } from './lookup.ts'

function toListRow(
  m: Mod,
  profile: Profile,
  byId: Record<number, { details?: ListRow['details'] } | undefined>,
  customCategories: readonly CustomCategory[],
): ListRow {
  const entry = (profile.entries ?? []).find((e) => e.key === m.key)
  const source = kindLabel(sourceKind(profile, m), {
    archive: i18n._(msg`Archive`),
    nexus: i18n._(msg`Nexus Mods`),
    github: i18n._(msg`GitHub`),
  })
  const details = byId[nexusIdOf(profile, m)]?.details
  const customById = customCategoryById(customCategories)
  const categoryLabel = resolvedCategoryLabel(
    entry?.categoryOverride,
    details?.category,
    customById,
  )
  const row: ListRow = {
    mod: m,
    added: entry?.added ?? '',
    pinned: Boolean(entry?.pinned),
    source,
    status: m.enabled ? i18n._(msg`Enabled`) : i18n._(msg`Off`),
    note: entry?.note ?? '',
    tags: [...(entry?.tags ?? [])],
    categoryOverride: entry?.categoryOverride ?? '',
    categoryLabel,
  }
  if (details) {
    row.details = details
  }
  return row
}

export { toListRow }
