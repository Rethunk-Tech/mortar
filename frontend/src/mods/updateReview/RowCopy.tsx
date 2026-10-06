import { useLingui } from '@lingui/react/macro'
import { Box, Tooltip, Typography } from '@mui/material'
import { TriangleAlert } from 'lucide-react'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { ModNameLink } from '../ModNameLink.tsx'
import { Changes } from './Changes.tsx'

export function RowCopy({
  update,
  caution,
  notes,
  reportedElsewhere,
  riskyChangelog,
}: {
  update: Update
  caution: string
  notes: string[]
  reportedElsewhere: boolean
  riskyChangelog: boolean
}) {
  const { t } = useLingui()
  return (
    <Box sx={{ minWidth: 0, display: 'flex', flexDirection: 'column', gap: '3px' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, minWidth: 0 }}>
        <Typography sx={{ fontSize: 16, fontWeight: 600, overflowWrap: 'anywhere', minWidth: 0 }}>
          {update.unofficial ? (
            t`Unofficial update available: ${update.version}`
          ) : (
            <ModNameLink id={update.id} modKey={update.key} name={update.name} wrap={true} />
          )}
        </Typography>
        {riskyChangelog ? (
          <Tooltip title={t`This update's notes mention breaking changes or new requirements`}>
            <Box
              component="span"
              sx={{ display: 'inline-flex', flexShrink: 0, color: 'warning.main' }}
            >
              <TriangleAlert size={16} aria-hidden={true} />
            </Box>
          </Tooltip>
        ) : null}
      </Box>
      {notes.length > 0 ? (
        <Typography
          sx={{
            alignSelf: 'flex-start',
            px: 1,
            borderRadius: '10px',
            bgcolor: 'var(--mortar-hairline-muted)',
            fontSize: 12,
          }}
        >
          {notes.join(' · ')}
        </Typography>
      ) : null}
      {caution ? (
        <Typography sx={{ fontSize: 12, color: 'warning.main', overflowWrap: 'anywhere' }}>
          {caution}
        </Typography>
      ) : null}
      {reportedElsewhere ? (
        <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
          {t`Reported by ${update.source}`}
        </Typography>
      ) : null}
      <Changes update={update} />
    </Box>
  )
}
