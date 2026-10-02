import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { Copy, TriangleAlert } from 'lucide-react'
import { useProfiles } from '../profiles/store.ts'
import { IconAction } from '../shell/IconAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useDescribe, useDescribeDrift } from './describe.ts'
import { DriftButtons, FixButton } from './problemFixButtons.tsx'
import { isInfoRow, type ProblemSectionId, problemSections, type Row } from './problemGroups.ts'
import { useMods } from './store.ts'
import { useLoadProblemsOnFocus } from './useLoadProblemsOnFocus.ts'

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

function ProblemRow({ row }: { row: Row }) {
  const info = isInfoRow(row)
  const { text, note: authorNote } = useRowText()(row)
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
      </Box>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, justifyContent: 'flex-end' }}>
        {row.kind === 'drift' ? <DriftButtons drift={row.drift} /> : <FixButton problem={row} />}
      </Box>
    </Box>
  )
}

function CleanupRow({ cleanup }: { cleanup: { key: string; uniqueId: string; name: string } }) {
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
          {t`${cleanup.name || cleanup.uniqueId}: Not needed by any enabled mod`}
        </Typography>
      </Box>
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
    </Box>
  )
}

// useOpenProblems is the open profile's problems, or null while they load, never the profile shown before.
function useOpenProblems() {
  const openId = useProfiles((s) => s.openId)
  return useMods((s) => (s.problemsFor === openId ? s.problems : null))
}

// ProblemActions sits in the profile's tab row while the Problems tab is open, like the Console's log actions.
export function ProblemActions() {
  const { t } = useLingui()
  const result = useOpenProblems()
  const sectionTitle = useSectionTitle()
  const rowText = useRowText()
  const sections = result === null ? [] : problemSections(result)
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
              ...section.rows.map((row) => {
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

  if (result === null) {
    return (
      <Typography sx={{ px: 2, py: 2, fontSize: 14, color: 'text.secondary' }}>
        {t`Checking the mods for problems…`}
      </Typography>
    )
  }

  const sections = problemSections(result)
  const cleanup = result.cleanup ?? []
  const empty = sections.length === 0 && cleanup.length === 0 && !result.unknown

  return (
    <Box
      sx={{
        flex: 1,
        minHeight: 0,
        overflowY: 'auto',
        px: 2,
        py: 2,
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
      }}
    >
      {empty ? (
        <Typography
          sx={{ fontSize: 14, color: 'text.secondary' }}
        >{t`No problems found.`}</Typography>
      ) : (
        sections.map((section) => (
          <Box key={section.id}>
            <Typography sx={{ mb: 1, fontSize: 13, fontWeight: 600, color: 'text.secondary' }}>
              {sectionTitle(section.id)}
            </Typography>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              {section.rows.map((row) => (
                <ProblemRow key={JSON.stringify(row)} row={row} />
              ))}
            </Box>
          </Box>
        ))
      )}
      {cleanup.length === 0 ? null : (
        <Box>
          <Typography sx={{ mb: 1, fontSize: 13, fontWeight: 600, color: 'text.secondary' }}>
            {t`Cleanup`}
          </Typography>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {cleanup.map((item) => (
              <CleanupRow key={item.key} cleanup={item} />
            ))}
          </Box>
        </Box>
      )}
      {result.unknown ? (
        <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
          {t`Some checks could not run without a connection, so more problems may show up later.`}
        </Typography>
      ) : null}
    </Box>
  )
}
