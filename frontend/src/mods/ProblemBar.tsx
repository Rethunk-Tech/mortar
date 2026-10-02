import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { TriangleAlert } from 'lucide-react'
import { useTab } from '../game/tab.ts'
import { useDescribe, useDescribeDrift } from './describe.ts'
import { problemsOf } from './lookup.ts'
import { driftRows, isInfoRow } from './problemGroups.ts'
import { useMods } from './store.ts'
import { useLoadProblemsOnFocus } from './useLoadProblemsOnFocus.ts'

// One row on the Mods tab: how many problems, the first warning's text, and the way to the Problems tab, which
// lists them in full. A list here pushed the mods out of view on large profiles.
export function ProblemBar() {
  const { t } = useLingui()
  const describe = useDescribe()
  const describeDrift = useDescribeDrift()
  const setTab = useTab((s) => s.setTab)
  const result = useMods((s) => s.problems)
  const problems = [...problemsOf(result), ...driftRows(result)]
  useLoadProblemsOnFocus()
  if (result === null) {
    return (
      <Typography sx={{ mx: 2, mt: 1.25, fontSize: 13, color: 'text.secondary' }}>
        {t`Checking the mods for problems…`}
      </Typography>
    )
  }
  if (problems.length === 0 && !result.unknown) {
    return null
  }
  const first = problems.find((p) => !isInfoRow(p)) ?? problems[0]
  const info = first === undefined || isInfoRow(first)
  const headline =
    problems.length === 0
      ? t`Some checks could not run without a connection, so more problems may show up later.`
      : plural(problems.length, { one: '# problem', other: '# problems' })
  const detail =
    first === undefined ? '' : first.kind === 'drift' ? describeDrift(first.drift) : describe(first)
  return (
    <ButtonBase
      onClick={() => setTab('problems')}
      sx={{
        mx: 2,
        mt: 1.25,
        flexShrink: 0,
        display: 'flex',
        alignItems: 'center',
        gap: 1.25,
        height: 38,
        pl: 1.5,
        pr: 0.75,
        textAlign: 'left',
        bgcolor: info ? 'rgba(56,189,248,0.12)' : 'rgba(243,180,22,0.14)',
        border: info ? '1px solid rgba(56,189,248,0.45)' : '1px solid rgba(243,180,22,0.5)',
        borderRadius: '6px',
      }}
    >
      <Box
        component="span"
        sx={{ display: 'flex', flexShrink: 0, color: info ? 'info.main' : 'warning.main' }}
      >
        <TriangleAlert size={16} aria-hidden={true} />
      </Box>
      <Typography sx={{ flexShrink: 0, fontSize: 14, fontWeight: 600 }}>{headline}</Typography>
      <Typography
        noWrap={true}
        sx={{ flex: 1, minWidth: 0, fontSize: 14, color: 'text.secondary' }}
      >
        {detail}
      </Typography>
      <Typography component="span" sx={{ flexShrink: 0, fontSize: 14, fontWeight: 600, px: 1 }}>
        {t`Open Problems`}
      </Typography>
    </ButtonBase>
  )
}
