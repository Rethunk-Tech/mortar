import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { TriangleAlert } from 'lucide-react'
import { useDescribe, useDescribeDrift } from './describe.ts'
import { DriftButtons, FixButton } from './problemFixButtons.tsx'
import { isInfoRow, type ProblemSectionId, problemSections, type Row } from './problemGroups.ts'
import { useMods } from './store.ts'
import { useLoadProblemsOnFocus } from './useLoadProblemsOnFocus.ts'

function SectionHeading({ id }: { id: ProblemSectionId }) {
  const { t } = useLingui()
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
    default:
      return ''
  }
}

function ProblemRow({ row }: { row: Row }) {
  const describe = useDescribe()
  const describeDrift = useDescribeDrift()
  const info = isInfoRow(row)
  const authorNote = row.kind === 'missing' && row.missing.listed ? row.missing.note.trim() : ''
  let text = row.kind === 'drift' ? describeDrift(row.drift) : describe(row)
  if (authorNote !== '' && text.endsWith(`: ${authorNote}`)) {
    text = text.slice(0, -(authorNote.length + 2))
  }
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

export function ProblemsTab() {
  const { t } = useLingui()
  const result = useMods((s) => s.problems)
  useLoadProblemsOnFocus()

  if (result === null) {
    return (
      <Typography sx={{ px: 2, py: 2, fontSize: 14, color: 'text.secondary' }}>
        {t`Checking the mods for problems…`}
      </Typography>
    )
  }

  const sections = problemSections(result)
  const empty = sections.length === 0 && !result.unknown

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
              <SectionHeading id={section.id} />
            </Typography>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              {section.rows.map((row) => (
                <ProblemRow key={JSON.stringify(row)} row={row} />
              ))}
            </Box>
          </Box>
        ))
      )}
      {result.unknown ? (
        <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
          {t`Some checks could not run without a connection, so more problems may show up later.`}
        </Typography>
      ) : null}
    </Box>
  )
}
