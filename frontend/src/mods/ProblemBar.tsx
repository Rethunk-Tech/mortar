import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Skeleton, Typography } from '@mui/material'
import { TriangleAlert } from 'lucide-react'
import { useTab } from '../game/tab.ts'
import { problemsLabel } from '../i18n/counts.ts'
import { useProfiles } from '../profiles/store.ts'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { useBadges } from './badges.ts'
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
  const openId = useProfiles((s) => s.openId)
  const problems = [...problemsOf(result), ...driftRows(result)]
  const known = useBadges((s) => s.byProfile[openId ?? ''])
  useLoadProblemsOnFocus()
  if (result === null) {
    // A slow scan shows the profile's last known count rather than a bare grey bar.
    const lastCount = known === undefined ? 0 : known.problems + known.missing
    if (lastCount > 0) {
      return (
        <Box
          role="status"
          aria-busy={true}
          sx={{
            mx: 2,
            mt: 1.25,
            flexShrink: 0,
            display: 'flex',
            alignItems: 'center',
            gap: 1.25,
            height: 38,
            px: 1.5,
            borderRadius: '6px',
            border: '1px solid',
            borderColor: 'divider',
            color: 'text.secondary',
          }}
        >
          <TriangleAlert size={16} aria-hidden={true} />
          <Typography component="span" sx={{ fontSize: 14, fontWeight: 600 }}>
            {problemsLabel(lastCount)}
          </Typography>
          <Typography component="span" sx={{ fontSize: 14 }}>
            {t`Checking again…`}
          </Typography>
        </Box>
      )
    }
    return (
      <Skeleton
        variant="rounded"
        role="status"
        aria-label={t`Checking the mods for problems…`}
        height={38}
        sx={{ mx: 2, mt: 1.25, flexShrink: 0 }}
      />
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
      : problemsLabel(problems.length)
  let detail = ''
  if (first !== undefined) {
    detail = first.kind === 'drift' ? describeDrift(first.drift) : describe(first)
  }
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
        bgcolor: calloutFill(info ? 'info' : 'warning'),
        border: '1px solid',
        borderColor: calloutLine(info ? 'info' : 'warning'),
        borderRadius: '6px',
      }}
    >
      <Box
        component="span"
        sx={{ display: 'flex', flexShrink: 0, color: info ? 'info.main' : 'warning.main' }}
      >
        <TriangleAlert size={16} aria-hidden={true} />
      </Box>
      <Typography component="span" sx={{ flexShrink: 0, fontSize: 14, fontWeight: 600 }}>
        {headline}
      </Typography>
      <Typography
        component="span"
        noWrap={true}
        title={detail}
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
