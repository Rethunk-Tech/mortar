import { useLingui } from '@lingui/react/macro'
import { Box, Chip, Link, Typography } from '@mui/material'
import type { Compat } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { compatOf, showCompatChip } from './compatChip.ts'
import { openPage } from './menu.ts'
import { heading } from './paper.ts'
import { useMods } from './store.ts'

const text = { fontSize: 13 } as const

export function CompatChip({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const row = compatOf(
    useMods((s) => s.problems?.compat),
    mod,
  )
  if (!showCompatChip(row?.status)) {
    return null
  }
  const labels: Record<string, string> = {
    optional: t`Optional`,
    unofficial: t`Unofficial`,
    broken: t`Broken`,
    obsolete: t`Obsolete`,
    abandoned: t`Abandoned`,
  }
  const status = (row?.status ?? '').toLowerCase()
  const label = labels[status] ?? status
  return <Chip size="small" label={label} title={row?.summary || label} />
}

export function CompatDetail({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const row = compatOf(
    useMods((s) => s.problems?.compat),
    mod,
  ) as Compat | undefined
  if (!showCompatChip(row?.status) || row === undefined) {
    return null
  }
  const unofficial =
    row.unofficialUrl === '' ? null : (
      <Link
        component="button"
        type="button"
        underline="hover"
        sx={{ ...text, textAlign: 'left' }}
        onClick={() => openPage(row.unofficialUrl)}
      >
        {t`Unofficial update`}
      </Link>
    )
  const replacement =
    row.replacement === '' ? null : (
      <Typography sx={text}>{t`Replacement: ${row.replacement}`}</Typography>
    )
  const broke =
    row.brokeIn === '' ? null : <Typography sx={text}>{t`Broke in ${row.brokeIn}`}</Typography>
  const summary = row.summary === '' ? null : <Typography sx={text}>{row.summary}</Typography>
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Typography sx={heading}>{t`Compatibility`}</Typography>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minHeight: 30, ...text }}>
        <CompatChip mod={mod} />
        {summary}
      </Box>
      {broke}
      {unofficial}
      {replacement}
    </Box>
  )
}
