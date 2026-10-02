import type { Result } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type { Drift } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
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
  (p.kind === 'broken' && p.broken.status === 'abandoned') ||
  (p.kind === 'missing' && p.missing.listed) ||
  p.kind === 'setting' ||
  (p.kind === 'runError' && !p.runError.severe)

export type ProblemSectionId =
  | 'missing'
  | 'conflicts'
  | 'broken'
  | 'runErrors'
  | 'drift'
  | 'duplicates'
  | 'settings'
  | 'cosmetic'
  | 'dismissed'

export interface ProblemSection {
  id: ProblemSectionId
  rows: (Row | DismissedRow)[]
}

export function problemSections(result: Result): ProblemSection[] {
  const sections: ProblemSection[] = [
    {
      id: 'missing',
      rows: (result.missing ?? []).map((missing): Row => ({ kind: 'missing', missing })),
    },
    {
      id: 'conflicts',
      rows: (result.assetConflicts ?? [])
        .filter((asset) => !asset.cosmetic)
        .map((asset): Row => ({ kind: 'asset', asset })),
    },
    {
      id: 'broken',
      rows: (result.broken ?? []).map((broken): Row => ({ kind: 'broken', broken })),
    },
    {
      id: 'runErrors',
      rows: (result.runErrors ?? []).map((runError): Row => ({ kind: 'runError', runError })),
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
      rows: (result.assetConflicts ?? [])
        .filter((asset) => asset.cosmetic)
        .map((asset): Row => ({ kind: 'asset', asset })),
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

export function assetFixUsesQuietButtons(cosmetic: boolean): boolean {
  return cosmetic
}

export function assetFixButtonStyle(cosmetic: boolean): {
  variant: 'contained' | 'outlined'
  color: 'warning' | 'inherit'
} {
  return assetFixUsesQuietButtons(cosmetic)
    ? { variant: 'outlined', color: 'inherit' }
    : { variant: 'contained', color: 'warning' }
}

export function problemCount(result: Result | null): number {
  if (!result) {
    return 0
  }
  return problemSections(result)
    .filter((s) => s.id !== 'cosmetic')
    .reduce((n, s) => n + s.rows.length, 0)
}
