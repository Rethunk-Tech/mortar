import { useLingui } from '@lingui/react/macro'
import { Box, Link, Typography } from '@mui/material'
import type { Compat } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { LinkedText } from './ModLinks.tsx'
import { openPage } from './menu.ts'

function CompatInfoRow({ row }: { row: Compat }) {
  const { t } = useLingui()
  const summary = row.summary === '' ? '' : ` · ${row.summary}`
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
        bgcolor: calloutFill('info'),
        border: '1px solid',
        borderColor: calloutLine('info'),
        borderRadius: '6px',
      }}
    >
      <Typography sx={{ fontSize: 14 }}>
        <LinkedText
          text={`${row.name}: ${row.status}${summary}`}
          links={[{ name: row.name, key: row.key, id: row.id }]}
        />
      </Typography>
      {unofficial}
      {replacement}
    </Box>
  )
}

export function CompatSection({ rows }: { rows: Compat[] }) {
  if (rows.length === 0) {
    return null
  }
  return (
    <Box>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        {rows.map((row) => (
          <CompatInfoRow key={`${row.key}/${row.id}`} row={row} />
        ))}
      </Box>
    </Box>
  )
}
