import { useLingui } from '@lingui/react/macro'
import { Box, Link, Typography } from '@mui/material'
import { alpha, useTheme } from '@mui/material/styles'
import type { Compat } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { openPage } from './menu.ts'

const INFO_FILL = 0.12
const INFO_LINE = 0.45

function CompatInfoRow({ row }: { row: Compat }) {
  const { t } = useLingui()
  const info = useTheme().palette.info.main
  const summary = row.summary === '' ? '' : ` — ${row.summary}`
  const unofficial =
    row.unofficialUrl === '' ? null : (
      <Link
        component="button"
        type="button"
        underline="hover"
        sx={{ fontSize: 13, textAlign: 'left' }}
        onClick={() => openPage(row.unofficialUrl)}
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
        bgcolor: alpha(info, INFO_FILL),
        border: `1px solid ${alpha(info, INFO_LINE)}`,
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
