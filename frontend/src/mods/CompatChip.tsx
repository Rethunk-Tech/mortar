import { useLingui } from '@lingui/react/macro'
import { Box, Chip, Link, Typography } from '@mui/material'
import type { Compat } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { space } from '../theme/density.ts'
import { compatOf, compatStatusLabel, showCompatChip } from './compatChip.ts'
import { openPage } from './menu.ts'
import { heading } from './paper.ts'
import { useMods } from './store.ts'

const text = { fontSize: 'base.fontSize' } as const

// The mod's compatibility row: SMAPI's list first, else Mortar's own known-broken list, read as a broken row.
function useCompatRow(mod: Mod): Compat | undefined {
  const listed = compatOf(
    useMods((s) => s.problems?.compat),
    mod,
  )
  const curated = compatOf(
    useMods((s) => s.problems?.broken),
    mod,
  )
  if (listed) {
    return listed
  }
  if (curated?.source !== MORTAR_LIST) {
    return undefined
  }
  return {
    key: curated.key,
    id: curated.id,
    name: curated.name,
    status: curated.status,
    summary: curated.summary ?? '',
    brokeIn: '',
    unofficialUrl: '',
    replacement: curated.replacement?.pageName ?? '',
  }
}

// The Source of a row from Mortar's own known-broken list (problems.mortarSiteName).
const MORTAR_LIST = 'Mortar'

export function CompatChip({ mod }: { mod: Mod }) {
  const row = useCompatRow(mod)
  if (!showCompatChip(row?.status)) {
    return null
  }
  const label = compatStatusLabel(row?.status)
  return <Chip size="small" label={label} title={row?.summary || label} />
}

export function CompatDetail({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const row = useCompatRow(mod)
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
      <Typography sx={text}>{t`Replacement: ${{ replacement: row.replacement }}`}</Typography>
    )
  const broke =
    row.brokeIn === '' ? null : (
      <Typography sx={text}>{t`Broke in ${{ brokeIn: row.brokeIn }}`}</Typography>
    )
  const summary = row.summary === '' ? null : <Typography sx={text}>{row.summary}</Typography>
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Typography sx={heading}>{t`Compatibility`}</Typography>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minHeight: 30, ...text }}>
        <CompatChip mod={mod} />
        {summary}
      </Box>
      {broke}
      {unofficial}
      {replacement}
    </Box>
  )
}
