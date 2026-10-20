import { useLingui } from '@lingui/react/macro'
import { Box, Link, Typography } from '@mui/material'
import type { Compat } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { space } from '../theme/density.ts'
import { LinkedText } from './ModLinks.tsx'
import { openPage } from './menu.ts'
import { DismissAdvice } from './problemFix/DismissAdvice.tsx'

function CompatInfoRow({
  row,
  dismissedToken,
}: {
  row: Compat
  dismissedToken?: string | undefined
}) {
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
        pl: space.pad,
        pr: 0.75,
        py: space.gap,
        fontSize: 14,
        bgcolor: calloutFill('info'),
        border: '1px solid',
        borderColor: calloutLine('info'),
        borderRadius: '6px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <Typography sx={{ fontSize: 14, flex: 1, minWidth: 0 }}>
          <LinkedText
            text={`${row.name}: ${row.status}${summary}`}
            links={[{ name: row.name, key: row.key, id: row.id }]}
          />
        </Typography>
        <DismissAdvice id={row.id} dismissedToken={dismissedToken} />
      </Box>
      {unofficial}
      {replacement}
    </Box>
  )
}

export function CompatSection({
  rows,
  dismissed,
}: {
  rows: Compat[]
  dismissed: { token: string; row: Compat }[]
}) {
  const { t } = useLingui()
  if (rows.length === 0 && dismissed.length === 0) {
    return null
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
      {rows.map((row) => (
        <CompatInfoRow key={`${row.key}/${row.id}`} row={row} />
      ))}
      {dismissed.length > 0 ? (
        <Typography sx={{ fontSize: 13, color: 'var(--mortar-ink-sec)', mt: space.gap }}>
          {t`Dismissed`}
        </Typography>
      ) : null}
      {dismissed.map((d) => (
        <CompatInfoRow key={d.token} row={d.row} dismissedToken={d.token} />
      ))}
    </Box>
  )
}
