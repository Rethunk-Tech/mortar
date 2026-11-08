import { msg } from '@lingui/core/macro'
import type { Result } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { formatDuration, type SlowStartup } from '../console/startupView.ts'
import { i18n } from '../i18n/index.ts'
import type { ReasonText } from '../i18n/reasonText.ts'
import { compatReportChunks } from './compatChip.ts'
import type { ConflictGroup } from './fileConflicts.ts'
import type { ProblemSectionId, Row } from './problemGroups.ts'
import { isDismissedRow } from './problemText.ts'

interface ProblemReportSection {
  title: string
  count: number
  whyKeys?: readonly string[]
  lines?: readonly string[]
}

interface WhyEvidence {
  packId?: string
  target?: string
}

function whyKeysOf(evidence: readonly WhyEvidence[] | null | undefined): string[] {
  const keys: string[] = []
  for (const item of evidence ?? []) {
    const key = (item.target ?? '').trim() || (item.packId ?? '').trim()
    if (key !== '') {
      keys.push(key)
    }
  }
  return keys
}

function formatProblemReport(
  sections: readonly ProblemReportSection[],
  harmlessLabel: string,
  harmlessCount: number,
): string {
  const lines = sections.flatMap((section) => {
    const head = `${section.title}: ${section.count}`
    const keys = (section.whyKeys ?? []).filter((key) => key.trim() !== '')
    const extra = (section.lines ?? []).filter((line) => line.trim() !== '')
    const rest = [...keys.map((key) => `  ${key}`), ...extra.map((line) => `  ${line}`)]
    return rest.length === 0 ? [head] : [head, ...rest]
  })
  lines.push(`${harmlessLabel}: ${harmlessCount}`)
  return lines.join('\n')
}

const copyChunk = (title: string, lines: string[]) =>
  lines.length === 0 ? [] : [{ title, count: lines.length, lines }]

function formatOpenCopy({
  locale,
  sectionTitle,
  rowText,
  reasonText,
  sections,
  cleanup,
  redundant,
  compat,
  slowRows,
  fileConflicts,
  replaced,
  harmlessCount,
  evidenceOf,
}: {
  locale: string
  sectionTitle: (id: ProblemSectionId) => string
  rowText: (row: Row) => { text: string; note: string }
  reasonText: ReasonText
  sections: { id: ProblemSectionId; rows: Parameters<typeof isDismissedRow>[0][] }[]
  cleanup: NonNullable<Result['cleanup']>
  redundant: { text?: string; name: string; reason: string }[]
  compat: NonNullable<Result['compat']>
  slowRows: SlowStartup[]
  fileConflicts: ConflictGroup[]
  replaced: NonNullable<Result['replaced']>
  harmlessCount: number
  evidenceOf: {
    get: (asset: Extract<Row, { kind: 'asset' }>['asset']) => WhyEvidence[] | null | undefined
  }
}): string {
  const lineOf = (entry: Parameters<typeof isDismissedRow>[0]) => {
    const row = isDismissedRow(entry) ? entry.row : entry
    const { text: sentence, note } = rowText(row)
    return note === '' ? sentence : `${sentence} ${note}`
  }
  return formatProblemReport(
    [
      ...sections.map((section) => ({
        title: sectionTitle(section.id),
        count: section.rows.length,
        whyKeys: section.rows.flatMap((entry) => {
          const row = isDismissedRow(entry) ? entry.row : entry
          return row.kind === 'asset' ? whyKeysOf(evidenceOf.get(row.asset)) : []
        }),
        lines: section.rows.map(lineOf),
      })),
      ...copyChunk(
        i18n._(msg`Cleanup`),
        cleanup.map((item) => {
          const who = item.name.trim() === '' ? i18n._(msg`Unknown mod`) : item.name
          const reason = reasonText(
            item.code,
            item.params,
            item.reason || i18n._(msg`Not needed by any enabled mod`),
          )
          return `${who}: ${reason}`
        }),
      ),
      ...copyChunk(
        i18n._(msg`Redundant`),
        redundant.map((item) => item.text ?? `${item.name}: ${item.reason}`),
      ),
      ...compatReportChunks(compat, i18n._(msg`Compatibility`)),
      ...copyChunk(
        i18n._(msg`Slow startup`),
        slowRows.map((row) => {
          const time = formatDuration(row.ms, locale)
          return row.kind === 'pack'
            ? i18n._(
                msg`${{ name: row.name }} adds ${time} to startup (through ${{ framework: row.framework }})`,
              )
            : i18n._(msg`${{ name: row.name }} adds ${time} to startup`)
        }),
      ),
      ...copyChunk(
        i18n._(msg`File conflicts`),
        fileConflicts.map((fileConflict) => lineOf({ kind: 'fileConflict', fileConflict })),
      ),
      ...copyChunk(
        i18n._(msg`Replaced`),
        replaced.map((file) => file.path),
      ),
    ],
    i18n._(msg`Harmless`),
    harmlessCount,
  )
}

export { formatOpenCopy, formatProblemReport, whyKeysOf }
