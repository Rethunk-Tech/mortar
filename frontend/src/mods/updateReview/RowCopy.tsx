import { useLingui } from '@lingui/react/macro'
import { Box, Chip, Typography } from '@mui/material'
import { TriangleAlert } from 'lucide-react'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { Tooltip } from '../../shell/Tooltip.tsx'
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
        <Typography
          sx={{
            fontSize: 'large.fontSize',
            fontWeight: 600,
            overflowWrap: 'anywhere',
            minWidth: 0,
          }}
        >
          {update.unofficial ? (
            t`Unofficial update available: ${{ version: update.version }}`
          ) : (
            <ModNameLink id={update.id} modKey={update.key} name={update.name} wrap={true} />
          )}
        </Typography>
        {riskyChangelog ? (
          <Tooltip title={t`This update's notes mention breaking changes or new requirements`}>
            <Box
              component="span"
              role="img"
              tabIndex={0}
              aria-label={t`This update's notes mention breaking changes or new requirements`}
              sx={{ display: 'inline-flex', flexShrink: 0, color: 'warning.main' }}
            >
              <TriangleAlert size={16} aria-hidden={true} />
            </Box>
          </Tooltip>
        ) : null}
      </Box>
      {notes.length > 0 ? (
        <Chip
          size="small"
          label={notes.join(' · ')}
          sx={{
            alignSelf: 'flex-start',
            bgcolor: 'var(--mortar-hairline-muted)',
            fontSize: 'small.fontSize',
          }}
        />
      ) : null}
      {caution ? (
        <Typography
          sx={{ fontSize: 'small.fontSize', color: 'warning.main', overflowWrap: 'anywhere' }}
        >
          {caution}
        </Typography>
      ) : null}
      {update.fixesKnownBroken ? (
        <Typography sx={{ fontSize: 'small.fontSize', color: 'success.main' }}>
          {t`Mortar's known-broken list flags the installed version; this one is past it`}
        </Typography>
      ) : null}
      {reportedElsewhere ? (
        <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
          {t`Reported by ${{ source: update.source }}`}
        </Typography>
      ) : null}
      <Changes update={update} />
    </Box>
  )
}
