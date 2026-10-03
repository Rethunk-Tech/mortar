import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  ButtonBase,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { ChevronDown, ChevronRight, Copy, ShieldCheck, TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { download, type Want } from '../queue/actions.ts'
import { useSettings } from '../settings/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { ConflictWhy } from './ConflictWhy.tsx'
import { useDescribe, useDescribeDrift } from './describe.ts'
import { DriftButtons, FixButton } from './problemFixButtons.tsx'
import {
  type DismissedRow,
  isInfoRow,
  type ProblemSectionId,
  problemSections,
  type Row,
} from './problemGroups.ts'
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
      role="alert"
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: 1.25,
        flexShrink: 0,
        pl: 1.5,
        pr: 0.75,
        py: 1,
        fontSize: 14,
        bgcolor: info ? 'rgba(56,189,248,0.12)' : 'rgba(243,180,22,0.14)',
        border: info ? '1px solid rgba(56,189,248,0.45)' : '1px solid rgba(243,180,22,0.5)',
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
          color: info ? 'info.main' : 'warning.main',
        }}
      >
        <TriangleAlert size={16} aria-hidden={true} />
      </Box>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography sx={{ fontSize: 14, whiteSpace: 'normal', wordBreak: 'break-word' }}>
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

function CleanupRow({
  cleanup,
}: {
  cleanup: { key: string; uniqueId: string; name: string; reason?: string }
}) {
  const { t } = useLingui()
  const remove = useMods((s) => s.remove)
  const mod = useMods((s) => s.mods.find((candidate) => candidate.key === cleanup.key))
  return (
    <Box
      role="alert"
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: 1.25,
        flexShrink: 0,
        pl: 1.5,
        pr: 0.75,
        py: 1,
        fontSize: 14,
        bgcolor: 'rgba(56,189,248,0.12)',
        border: '1px solid rgba(56,189,248,0.45)',
        borderRadius: '6px',
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography sx={{ fontSize: 14, whiteSpace: 'normal', wordBreak: 'break-word' }}>
          {t`${cleanup.name || cleanup.uniqueId}: ${cleanup.reason || 'Not needed by any enabled mod'}`}
        </Typography>
      </Box>
      <DisabledReason title={t`This mod is no longer in the profile.`} disabled={mod === undefined}>
        <Button
          size="small"
          disabled={mod === undefined}
          onClick={() => {
            if (mod !== undefined) {
              remove(mod).catch(reportUnexpected)
            }
          }}
        >
          {t`Remove`}
        </Button>
      </DisabledReason>
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
  action?: { label: string; onClick: () => void }
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
          onClick={() => setOpen(!open)}
          aria-expanded={open}
          sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 1, borderRadius: '4px' }}
        >
          {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          {heading}
          {action ? (
            <Button
              size="small"
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
            <Button size="small" onClick={action.onClick} sx={{ ml: 1, height: 26 }}>
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
    action?: { label: string; onClick: () => void }
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

// ProblemActions sits in the profile's tab row while the Problems tab is open, like the Console's log actions.
export function ProblemActions() {
  const { t } = useLingui()
  const result = useOpenProblems()
  const sectionTitle = useSectionTitle()
  const rowText = useRowText()
  const sections =
    result === null ? [] : problemSections(result).filter((section) => section.id !== 'dismissed')
  const cleanup = result?.cleanup ?? []
  return (
    <IconAction
      label={t`Copy all problems`}
      icon={<Copy size={16} />}
      disabled={sections.length === 0 && cleanup.length === 0}
      onClick={() => {
        const text = [
          ...sections.map((section) =>
            [
              sectionTitle(section.id),
              ...section.rows.map((entry) => {
                const row = isDismissedRow(entry) ? entry.row : entry
                const { text: line, note } = rowText(row)
                return note === '' ? `- ${line}` : `- ${line}\n  ${note}`
              }),
            ].join('\n'),
          ),
          ...(cleanup.length === 0
            ? []
            : [
                [
                  t`Cleanup`,
                  ...cleanup.map(
                    (item) =>
                      `- ${t`${item.name || item.uniqueId}: Not needed by any enabled mod`}`,
                  ),
                  ...cleanup.map(
                    (item) =>
                      `- ${item.name || item.uniqueId}: ${t`Not needed by any enabled mod`}`,
                  ),
                ].join('\n'),
              ]),
        ]
          .filter((value, index, values) => values.indexOf(value) === index)
          .join('\n\n')
        Clipboard.SetText(text).then(
          () => useToasts.getState().push({ kind: 'success', title: t`Problems copied` }),
          reportUnexpected,
        )
      }}
    />
  )
}

export function ProblemsTab() {
  const { t } = useLingui()
  const result = useOpenProblems()
  const sectionTitle = useSectionTitle()
  useLoadProblemsOnFocus()
  const removeMany = useMods((s) => s.removeMany)
  const dismissAsset = useMods((s) => s.dismissAsset)
  const [confirmCleanup, setConfirmCleanup] = useState(false)
  const cosmeticConflicts = useSettings((s) => s.cosmeticConflicts)

  if (result === null) {
    return <LoadingRow>{t`Checking the mods for problems…`}</LoadingRow>
  }

  const sections = problemSections(result)
  const cleanup = result.cleanup ?? []
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
    sections.filter((section) => section.id !== 'dismissed').length === 0 &&
    cleanup.length === 0 &&
    !result.unknown

  const sectionExtras = (section: (typeof sections)[number]) => {
    if (section.id === 'missing' && installable.length > 0) {
      return {
        action: {
          label: t`Add all ${installable.length}`,
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
            download(wants).catch(reportUnexpected)
          },
        },
      }
    }
    if (section.id === 'cosmetic') {
      return {
        action: {
          label: t`Dismiss all`,
          onClick: () => {
            for (const row of section.rows) {
              if (!isDismissedRow(row) && row.kind === 'asset') {
                dismissAsset(row.asset).catch(reportUnexpected)
              }
            }
          },
        },
      }
    }
    return {}
  }

  return (
    <Box
      sx={{
        flex: 1,
        minHeight: 0,
        overflowY: 'auto',
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
      {cleanup.length === 0 ? null : (
        <Box>
          <Box sx={{ mb: 1, display: 'flex', alignItems: 'center' }}>
            <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary' }}>
              {t`Cleanup`}
            </Typography>
            <Button size="small" sx={{ ml: 1, height: 26 }} onClick={() => setConfirmCleanup(true)}>
              {t`Remove all`}
            </Button>
          </Box>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {cleanup.map((item) => (
              <CleanupRow key={item.key} cleanup={item} />
            ))}
          </Box>
        </Box>
      )}
      <Dialog open={confirmCleanup} onClose={() => setConfirmCleanup(false)}>
        <DialogTitle>{t`Remove all ${cleanup.length} mods from this profile?`}</DialogTitle>
        <DialogContent>{t`This change can be undone from History.`}</DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmCleanup(false)}>{t`Cancel`}</Button>
          <Button
            color="error"
            variant="contained"
            onClick={() => {
              setConfirmCleanup(false)
              const mods = cleanup
                .map((item) => useMods.getState().mods.find((mod) => mod.key === item.key))
                .filter((mod): mod is NonNullable<typeof mod> => mod !== undefined)
              removeMany(mods).catch(reportUnexpected)
            }}
          >
            {t`Remove all`}
          </Button>
        </DialogActions>
      </Dialog>
      {result.unknown ? (
        <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
          {t`Some checks could not run without a connection, so more problems may show up later.`}
        </Typography>
      ) : null}
    </Box>
  )
}
