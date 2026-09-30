export interface GameModRow {
  name: string
  version?: string
  source?: string
  status?: string
  reason?: string
  disabled?: boolean
}

export const willImport = (status: string | undefined) =>
  status !== 'skipped' && status !== 'failed'

export const withSwitchedOff = (name: string, disabled: boolean, note: string) =>
  disabled ? `${name} (${note})` : name

export const formatPreviewRow = (row: GameModRow, switchedOff: string) => {
  if (!willImport(row.status)) {
    return [row.name, row.reason].filter((p) => p !== undefined && p !== '').join(' · ')
  }
  return [withSwitchedOff(row.name, row.disabled === true, switchedOff), row.version, row.source]
    .filter((p) => p !== undefined && p !== '')
    .join(' · ')
}

export const formatOutcomeDetail = (
  outcomes: { name: string; status: string; reason?: string }[],
) =>
  outcomes
    .filter((o) => o.status === 'skipped' || o.status === 'failed')
    .map((o) => [o.name, o.reason].filter((p) => p !== undefined && p !== '').join(' · '))
    .join('\n')
