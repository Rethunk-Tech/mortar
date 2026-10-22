import type { ReasonText } from '../i18n/reasonText.ts'

export interface GameModRow {
  name: string
  version?: string
  source?: string
  status?: string
  reason?: string
  code?: string
  params?: { [_ in string]?: string } | null
  disabled?: boolean
}

export const willImport = (status: string | undefined) =>
  status !== 'skipped' && status !== 'failed'

export const withSwitchedOff = (name: string, disabled: boolean, note: string) =>
  disabled ? `${name} (${note})` : name

export const formatPreviewRow = (row: GameModRow, switchedOff: string, reasonText: ReasonText) => {
  if (!willImport(row.status)) {
    return [row.name, row.reason && reasonText(row.code, row.params, row.reason)]
      .filter((p) => p !== undefined && p !== '')
      .join(' · ')
  }
  return [withSwitchedOff(row.name, row.disabled === true, switchedOff), row.version, row.source]
    .filter((p) => p !== undefined && p !== '')
    .join(' · ')
}

export const formatOutcomeDetail = (outcomes: GameModRow[], reasonText: ReasonText) =>
  outcomes
    .filter((o) => o.status === 'skipped' || o.status === 'failed')
    .map((o) =>
      [o.name, o.reason && reasonText(o.code, o.params, o.reason)]
        .filter((p) => p !== undefined && p !== '')
        .join(' · '),
    )
    .join('\n')
