import { useLingui } from '@lingui/react/macro'
import { Box, Link, Typography } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import type { Compat } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { reportUnexpected } from '../toasts/report.ts'

function CompatInfoRow({ row }: { row: Compat }) {
  const { t } = useLingui()
  const summary = row.summary === '' ? '' : ` — ${row.summary}`
  const unofficial =
    row.unofficialUrl === '' ? null : (
      <Link
        component="button"
        type="button"
        underline="hover"
        sx={{ fontSize: 13, textAlign: 'left' }}
        onClick={() => Browser.OpenURL(row.unofficialUrl).catch(reportUnexpected)}
      >
        {t`Unofficial update`}
      </Link>
    )
  const replacement =
    row.replacement === '' ? null : (
      <Typography sx={{ fontSize: 13 }}>{t`Replacement: ${row.replacement}`}</Typography>
    )
  return (
    <Box
      role="status"
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 0.5,
        pl: 1.5,
        pr: 0.75,
        py: 1,
        fontSize: 14,
        bgcolor: 'rgba(56,189,248,0.12)',
        border: '1px solid rgba(56,189,248,0.45)',
        borderRadius: '6px',
      }}
    >
      <Typography sx={{ fontSize: 14 }}>{`${row.name}: ${row.status}${summary}`}</Typography>
      {unofficial}
      {replacement}
    </Box>
  )
}

export function CompatSection({ rows }: { rows: Compat[] }) {
  const { t } = useLingui()
  if (rows.length === 0) {
    return null
  }
  return (
    <Box>
      <Typography sx={{ mb: 1, fontSize: 13, fontWeight: 600, color: 'text.secondary' }}>
        {t`Compatibility`}
      </Typography>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        {rows.map((row) => (
          <CompatInfoRow key={`${row.key}/${row.uniqueId}`} row={row} />
        ))}
      </Box>
    </Box>
  )
}
