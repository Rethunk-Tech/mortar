import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonBase, Typography } from '@mui/material'
import { alpha } from '@mui/material/styles'

const WARN_FILL = 0.14
const WARN_LINE = 0.5

import { Clipboard } from '@wailsio/runtime'
import {
  ChevronDown,
  ChevronRight,
  Copy,
  Info,
  Map as MapIcon,
  ShieldCheck,
  TriangleAlert,
} from 'lucide-react'
import { useState } from 'react'
import { useTab } from '../game/tab.ts'
import { useProfiles } from '../profiles/store.ts'
import { download, type Want } from '../queue/actions.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { AssetMapDialog } from './AssetMapDialog.tsx'
import { CleanupSection } from './CleanupSection.tsx'
import { CompatSection } from './CompatSection.tsx'
import { ConflictWhy } from './ConflictWhy.tsx'
import { compatReportChunks } from './compatChip.ts'
import { useDescribe, useDescribeDrift } from './describe.ts'
import { DriftButtons, FixButton } from './problemFixButtons.tsx'
import {
  type DismissedRow,
  isInfoRow,
  type ProblemSectionId,
  problemSections,
  type Row,
} from './problemGroups.ts'
import { formatProblemReport, whyKeysOf } from './problemReport.ts'
import { useRedundantReason } from './redundantReason.ts'
import { CheckTimings, SlowStartupSection } from './SlowStartupSection.tsx'
import { useMods } from './store.ts'
import { useLoadProblemsOnFocus } from './useLoadProblemsOnFocus.ts'

const isDismissedRow = (row: Row | DismissedRow): row is DismissedRow => 'row' in row

function useSectionTitle() {
  const { t } = useLingui()
  return (id: ProblemSectionId) => {
    switch (id) {
      case 'missing':
        return t`Missing requirements`
      case 'conflicts':
        return t`Conflicts`
      case 'broken':
        return t`Broken or outdated mods`
      case 'runErrors':
        return t`Errors in the last run`
      case 'drift':
        return t`Changed outside Mortar`
      case 'duplicates':
        return t`Duplicates`
      case 'settings':
        return t`Settings`
      case 'cosmetic':
        return t`Cosmetic or harmless`
      case 'dismissed':
        return t`Dismissed`
      default:
        return ''
    }
  }
}

// useRowText is a row's sentence and the author's note shown under it, shared by the list and Copy all.
function useRowText() {
  const describe = useDescribe()
  const describeDrift = useDescribeDrift()
  return (row: Row) => {
    let note = ''
    if (row.kind === 'missing' && row.missing.listed) {
      note = row.missing.note.trim()
    } else if (row.kind === 'setting') {
      note = row.setting.description.trim()
    }
    let text = row.kind === 'drift' ? describeDrift(row.drift) : describe(row)
    if (note !== '' && text.endsWith(`: ${note}`)) {
      text = text.slice(0, -(note.length + 2))
    }
    return { text, note }
  }
}

function ProblemRow({ row, dismissed }: { row: Row; dismissed?: DismissedRow }) {
  const { t } = useLingui()
  const info = isInfoRow(row)
  const { text, note: authorNote } = useRowText()(row)
  const [why, setWhy] = useState(false)
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: 1.25,
        flexShrink: 0,
        pl: 1.5,
        pr: 0.75,
        py: 1,
        fontSize: 14,
        bgcolor: (th) =>
          info ? 'var(--mortar-overlay-45)' : alpha(th.palette.warning.main, WARN_FILL),
        border: '1px solid',
        borderColor: (th) => (info ? 'transparent' : alpha(th.palette.warning.main, WARN_LINE)),
        borderRadius: '6px',
        ...(dismissed ? { opacity: 0.75 } : {}),
      }}
    >
      <Box
        component="span"
        sx={{
          display: 'flex',
          flexShrink: 0,
          mt: 0.25,
          color: info ? 'text.secondary' : 'warning.main',
        }}
      >
        {info ? (
          <Info size={16} aria-hidden={true} />
        ) : (
          <TriangleAlert size={16} aria-hidden={true} />
        )}
      </Box>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography
          sx={{ fontSize: 14, whiteSpace: 'normal', wordBreak: 'break-word' }}
          {...(row.kind === 'missing'
            ? {
                component: ButtonBase,
                onClick: () =>
                  useTab.getState().revealLoadOrder(row.missing.uniqueId, row.missing.dependentId),
                sx: {
                  fontSize: 14,
                  whiteSpace: 'normal',
                  wordBreak: 'break-word',
                  textAlign: 'left',
                  borderRadius: '4px',
                  textDecoration: 'underline',
                  textDecorationColor: 'var(--mortar-hairline-35)',
                },
              }
            : {})}
        >
          {text}
        </Typography>
        {authorNote === '' ? null : (
          <Typography sx={{ mt: 0.5, fontSize: 13, color: 'text.secondary', whiteSpace: 'normal' }}>
            {authorNote}
          </Typography>
        )}
        {row.kind === 'asset' ? (
          <Box sx={{ mt: 0.75 }}>
            <ButtonBase
              onClick={() => setWhy(!why)}
              aria-expanded={why}
              sx={{ display: 'flex', alignItems: 'center', gap: 0.5, borderRadius: '4px' }}
            >
              {why ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary' }}>
                {t`Why?`}
              </Typography>
            </ButtonBase>
            {why ? <ConflictWhy asset={row.asset} /> : null}
          </Box>
        ) : null}
      </Box>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, justifyContent: 'flex-end' }}>
        {row.kind === 'drift' ? (
          <DriftButtons drift={row.drift} />
        ) : (
          <FixButton problem={row} dismissedToken={dismissed?.token} />
        )}
      </Box>
    </Box>
  )
}

// useOpenProblems is the open profile's problems, or null while they load, never the profile shown before.
function useOpenProblems() {
  const openId = useProfiles((s) => s.openId)
  return useMods((s) => (s.problemsFor === openId ? s.problems : null))
}

// ProblemSection lists one kind of problem; harmless overlaps start collapsed so real problems lead.
function ProblemSection({
  title,
  rows,
  collapsible,
  action,
}: {
  title: string
  rows: (Row | DismissedRow)[]
  collapsible: boolean
  action?: { label: string; onClick: () => void; disabled?: boolean }
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(!collapsible)
  const count = rows.length
  const label = collapsible ? t`${title} · ${count}` : title
  const heading = (
    <Typography component="span" sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary' }}>
      {label}
    </Typography>
  )
  return (
    <Box>
      {collapsible ? (
        <ButtonBase
          component="div"
          onClick={() => setOpen(!open)}
          aria-expanded={open}
          sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 1, borderRadius: '4px' }}
        >
          {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          {heading}
          {action ? (
            <Button
              size="small"
              disabled={action.disabled}
              onClick={(e) => {
                e.stopPropagation()
                action.onClick()
              }}
              sx={{ ml: 1, height: 26 }}
            >
              {action.label}
            </Button>
          ) : null}
        </ButtonBase>
      ) : (
        <Box sx={{ mb: 1, display: 'flex', alignItems: 'center' }}>
          {heading}
          {action ? (
            <Button
              size="small"
              disabled={action.disabled}
              onClick={action.onClick}
              sx={{ ml: 1, height: 26 }}
            >
              {action.label}
            </Button>
          ) : null}
        </Box>
      )}
      {open ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          {rows.map((row) =>
            isDismissedRow(row) ? (
              <ProblemRow key={row.token} row={row.row} dismissed={row} />
            ) : (
              <ProblemRow key={JSON.stringify(row)} row={row} />
            ),
          )}
        </Box>
      ) : null}
    </Box>
  )
}

function renderProblemSections(
  sections: ReturnType<typeof problemSections>,
  cosmeticConflicts: string,
  sectionTitle: (id: ProblemSectionId) => string,
  extras: (section: ReturnType<typeof problemSections>[number]) => {
    action?: { label: string; onClick: () => void; disabled?: boolean }
  },
) {
  return sections.map((section) => {
    if (section.id === 'cosmetic' && cosmeticConflicts === 'hidden') {
      return null
    }
    const cosmeticOpen = cosmeticConflicts === 'expanded'
    return (
      <ProblemSection
        key={section.id}
        title={sectionTitle(section.id)}
        rows={section.rows}
        collapsible={(section.id === 'cosmetic' && !cosmeticOpen) || section.id === 'dismissed'}
        {...extras(section)}
      />
    )
  })
}

function dismissCosmetic(
  sections: ReturnType<typeof problemSections>,
  dismissAsset: ReturnType<typeof useMods.getState>['dismissAsset'],
) {
  const cosmetic = sections.find((section) => section.id === 'cosmetic')
  for (const row of cosmetic?.rows ?? []) {
    if (!isDismissedRow(row) && row.kind === 'asset') {
      dismissAsset(row.asset).catch(reportUnexpected)
    }
  }
}

function OfflineChecksNote() {
  const { t } = useLingui()
  return (
    <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
      {t`Some checks could not run without a connection, so more problems may show up later.`}
    </Typography>
  )
}

// ProblemActions sits in the profile's tab row while the Problems tab is open, like the Console's log actions.
export function ProblemActions() {
  const { t } = useLingui()
  const result = useOpenProblems()
  const [mapOpen, setMapOpen] = useState(false)
  const sectionTitle = useSectionTitle()
  const rowText = useRowText()
  const sections = result === null ? [] : problemSections(result)
  const cleanup = result?.cleanup ?? []
  const compat = result?.compat ?? []
  const redundantReason = useRedundantReason()
  const redundant = (result?.redundant ?? []).map((item) => ({
    ...item,
    reason: redundantReason(item),
  }))
  const harmlessCount = (result?.assetConflicts ?? []).filter((asset) => asset.cosmetic).length
  const nothing =
    result === null ||
    (sections.length === 0 &&
      cleanup.length + redundant.length + compat.length === 0 &&
      harmlessCount === 0)
  return (
    <>
      <IconAction
        label={t`Asset map`}
        icon={<MapIcon size={16} />}
        onClick={() => setMapOpen(true)}
      />
      <AssetMapDialog open={mapOpen} onClose={() => setMapOpen(false)} />
      <IconAction
        label={t`Copy report`}
        icon={<Copy size={16} />}
        disabled={nothing}
        disabledTitle={t`No problems to copy.`}
        onClick={() => {
          const text = formatProblemReport(
            [
              ...sections.map((section) => ({
                title: sectionTitle(section.id),
                count: section.rows.length,
                whyKeys: section.rows.flatMap((entry) => {
                  const row = isDismissedRow(entry) ? entry.row : entry
                  return row.kind === 'asset' ? whyKeysOf(row.asset.evidence) : []
                }),
                lines: section.rows.map((entry) => {
                  const row = isDismissedRow(entry) ? entry.row : entry
                  const { text: sentence, note } = rowText(row)
                  return note === '' ? sentence : `${sentence} ${note}`
                }),
              })),
              ...(cleanup.length === 0
                ? []
                : [
                    {
                      title: t`Cleanup`,
                      count: cleanup.length,
                      lines: cleanup.map((item) => {
                        const who = item.name.trim() === '' ? t`Unknown mod` : item.name
                        const reason = item.reason || t`Not needed by any enabled mod`
                        return `${who}: ${reason}`
                      }),
                    },
                  ]),
              ...(redundant.length === 0
                ? []
                : [
                    {
                      title: t`Redundant`,
                      count: redundant.length,
                      lines: redundant.map((item) => `${item.name}: ${item.reason}`),
                    },
                  ]),
              ...compatReportChunks(compat, t`Compatibility`),
            ],
            t`Harmless`,
            harmlessCount,
          )
          Clipboard.SetText(text).then(
            () => useToasts.getState().push({ kind: 'success', title: t`Report copied` }),
            reportUnexpected,
          )
        }}
      />
    </>
  )
}

export function ProblemsTab() {
  const { t } = useLingui()
  const result = useOpenProblems()
  const sectionTitle = useSectionTitle()
  useLoadProblemsOnFocus()
  const dismissAsset = useMods((s) => s.dismissAsset)
  const [confirmDismissCosmetic, setConfirmDismissCosmetic] = useState(false)
  const [addingAll, runAddAll] = usePending()
  const cosmeticConflicts = useSettings((s) => gamePrefs(s).cosmeticConflicts)
  const redundantReason = useRedundantReason()

  if (result === null) {
    return <LoadingRow>{t`Checking the mods for problems…`}</LoadingRow>
  }
  const sections = problemSections(result)
  const cleanup = result.cleanup ?? []
  const compat = result.compat ?? []
  const redundant = (result.redundant ?? []).map((item) => ({
    ...item,
    reason: redundantReason(item),
  }))
  const installable =
    sections
      .find((section) => section.id === 'missing')
      ?.rows.filter(
        (row): row is Extract<Row, { kind: 'missing' }> =>
          !isDismissedRow(row) &&
          row.kind === 'missing' &&
          row.missing.listed &&
          row.missing.where !== null,
      ) ?? []
  const empty =
    sections.filter((s) => s.id !== 'dismissed').length === 0 &&
    cleanup.length + redundant.length + compat.length === 0 &&
    !result.unknown

  const sectionExtras = (section: (typeof sections)[number]) => {
    if (section.id === 'missing' && installable.length > 0) {
      return {
        action: {
          label: t`Add all ${installable.length}`,
          disabled: addingAll,
          onClick: () => {
            const wants: Want[] = installable.flatMap(({ missing }): Want[] => {
              const { where } = missing
              if (!where) {
                return []
              }
              return where.site === 'GitHub'
                ? [{ kind: 'dependency', repo: where.github, name: where.github }]
                : [
                    {
                      kind: 'dependency',
                      modId: where.pageId,
                      fileId: where.fileId,
                      latest: true,
                      name: where.pageName,
                      fileName: where.fileName,
                      version: where.version,
                    },
                  ]
            })
            runAddAll(() => download(wants))
          },
        },
      }
    }
    if (section.id === 'cosmetic') {
      return {
        action: {
          label: t`Dismiss all`,
          onClick: () => {
            setConfirmDismissCosmetic(true)
          },
        },
      }
    }
    return {}
  }

  return (
    <Box
      sx={{
        flexShrink: 0,
        px: 2,
        pt: 2,
        pb: 1.5,
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
      }}
    >
      {empty ? (
        <EmptyState
          icon={<ShieldCheck size={40} aria-hidden={true} />}
          title={t`No problems found`}
        >
          {t`Every mod has what it needs and nothing clashes.`}
        </EmptyState>
      ) : null}
      {renderProblemSections(sections, cosmeticConflicts, sectionTitle, sectionExtras)}
      <CompatSection rows={compat} />
      <SlowStartupSection />
      <CleanupSection
        cleanup={redundant}
        title={t`Redundant`}
        removeAll={redundant.every((item) => item.kind !== 'patches')}
      />
      <CleanupSection cleanup={cleanup} />
      <ConfirmDialog
        open={confirmDismissCosmetic}
        title={t`Dismiss all harmless overlaps?`}
        body={t`They move to Dismissed. Restore them from that section.`}
        confirmLabel={t`Dismiss all`}
        onCancel={() => setConfirmDismissCosmetic(false)}
        onConfirm={() => {
          setConfirmDismissCosmetic(false)
          dismissCosmetic(sections, dismissAsset)
        }}
      />
      {result.unknown ? <OfflineChecksNote /> : null}
      <CheckTimings timings={result.timings ?? []} />
    </Box>
  )
}
