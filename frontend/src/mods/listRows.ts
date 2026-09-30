import type { useLingui } from '@lingui/react/macro'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type { ListRow } from './listColumns.ts'
import { kindLabel, nexusIdOf, sourceKind } from './lookup.ts'

function toListRow(
  m: Mod,
  profile: Profile,
  byId: Record<number, { details?: ListRow['details'] } | undefined>,
  t: ReturnType<typeof useLingui>['t'],
): ListRow {
  const entry = (profile.entries ?? []).find((e) => e.key === m.key)
  const source = kindLabel(sourceKind(profile, m), {
    archive: t`Archive`,
    nexus: t`Nexus Mods`,
    github: t`GitHub`,
  })
  const details = byId[nexusIdOf(profile, m)]?.details
  const row: ListRow = {
    mod: m,
    added: entry?.added ?? '',
    pinned: Boolean(entry?.pinned),
    source,
    status: m.enabled ? t`Enabled` : t`Off`,
    note: entry?.note ?? '',
    tags: [...(entry?.tags ?? [])],
  }
  if (details) {
    row.details = details
  }
  return row
}

export { toListRow }
