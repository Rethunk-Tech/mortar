import type { AssetConflict } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/framework/models.ts'
import type { Result } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Drift } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Problem } from './lookup.ts'

export type Row = Problem | { kind: 'drift'; drift: Drift }
export interface DismissedRow {
  row: Problem
  token: string
}

export const driftRows = (result: Result | null): Row[] =>
  (result?.drift ?? []).map((drift) => ({ kind: 'drift' as const, drift }))

export const isInfoRow = (p: Row): boolean =>
  (p.kind === 'asset' && p.asset.kind === 'edit') ||
  (p.kind === 'duplicate' && p.duplicate.nexusOptional === true) ||
  (p.kind === 'broken' && p.broken.status === 'abandoned') ||
  (p.kind === 'missing' && p.missing.listed) ||
  p.kind === 'setting' ||
  (p.kind === 'runError' && !p.runError.severe)

export type ProblemSectionId =
  | 'missing'
  | 'conflicts'
  | 'broken'
  | 'damaged'
  | 'runErrors'
  | 'loadFailures'
  | 'pluginClashes'
  | 'drift'
  | 'duplicates'
  | 'settings'
  | 'cosmetic'
  | 'dismissed'

export interface ProblemSection {
  id: ProblemSectionId
  rows: (Row | DismissedRow)[]
}

// assetRows shows conflicts between the same mods, with the same outcome and fix, as one row: a pack that edits
// four seasonal maps the same way would otherwise fill four rows with one story.
export function assetRows(conflicts: AssetConflict[]): Row[] {
  const groups = new Map<string, AssetConflict[]>()
  for (const asset of conflicts) {
    const key = JSON.stringify([
      asset.kind,
      [...(asset.packIds ?? [])].sort(),
      asset.winnerName,
      asset.fixes ?? [],
    ])
    groups.set(key, [...(groups.get(key) ?? []), asset])
  }
  const rows: Row[] = []
  for (const [asset, ...siblings] of groups.values()) {
    if (asset !== undefined) {
      rows.push(
        siblings.length === 0 ? { kind: 'asset', asset } : { kind: 'asset', asset, siblings },
      )
    }
  }
  return rows
}

export function problemSections(result: Result): ProblemSection[] {
  const sections: ProblemSection[] = [
    {
      id: 'missing',
      rows: (result.missing ?? []).map((missing): Row => ({ kind: 'missing', missing })),
    },
    {
      id: 'conflicts',
      rows: assetRows((result.assetConflicts ?? []).filter((asset) => !asset.cosmetic)),
    },
    {
      id: 'broken',
      rows: (result.broken ?? []).map((broken): Row => ({ kind: 'broken', broken })),
    },
    {
      id: 'damaged',
      rows: (result.damaged ?? []).map((damaged): Row => ({ kind: 'damaged', damaged })),
    },
    {
      id: 'runErrors',
      rows: (result.runErrors ?? []).map((runError): Row => ({ kind: 'runError', runError })),
    },
    {
      id: 'loadFailures',
      rows: (result.loadFailures ?? []).map(
        (loadFailure): Row => ({ kind: 'loadFailure', loadFailure }),
      ),
    },
    {
      id: 'pluginClashes',
      rows: (result.pluginClashes ?? []).map(
        (pluginClash): Row => ({ kind: 'pluginClash', pluginClash }),
      ),
    },
    {
      id: 'drift',
      rows: driftRows(result).filter((r) => r.kind === 'drift'),
    },
    {
      id: 'duplicates',
      rows: (result.duplicates ?? []).map((duplicate): Row => ({ kind: 'duplicate', duplicate })),
    },
    {
      id: 'settings',
      rows: (result.settings ?? []).map((setting): Row => ({ kind: 'setting', setting })),
    },
    {
      id: 'cosmetic',
      rows: assetRows((result.assetConflicts ?? []).filter((asset) => asset.cosmetic)),
    },
    {
      id: 'dismissed',
      rows: (result.dismissed ?? []).flatMap((item): DismissedRow[] => {
        if (item.assetConflict) {
          return [{ token: item.token, row: { kind: 'asset', asset: item.assetConflict } }]
        }
        if (item.broken) {
          return [{ token: item.token, row: { kind: 'broken', broken: item.broken } }]
        }
        if (item.missing) {
          return [{ token: item.token, row: { kind: 'missing', missing: item.missing } }]
        }
        if (item.setting) {
          return [{ token: item.token, row: { kind: 'setting', setting: item.setting } }]
        }
        return []
      }),
    },
  ]
  return sections.filter((s) => s.rows.length > 0)
}

export function assetFixButtonStyle(cosmetic: boolean): {
  variant: 'contained' | 'outlined'
  color: 'warning' | 'inherit'
} {
  return cosmetic
    ? { variant: 'outlined', color: 'inherit' }
    : { variant: 'contained', color: 'warning' }
}
