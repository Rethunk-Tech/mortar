import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { TriangleAlert } from 'lucide-react'
import { useTab } from '../game/tab.ts'
import { useDescribe, useDescribeDrift } from './describe.ts'
import { problemsOf } from './lookup.ts'
import { DriftButtons, FixButton } from './problemFixButtons.tsx'
import { driftRows, isInfoRow } from './problemGroups.ts'
import { useMods } from './store.ts'
import { useLoadProblemsOnFocus } from './useLoadProblemsOnFocus.ts'

const ROW_HEIGHT = 38
const ROW_GAP = 6
const VISIBLE_ROWS = 3

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
  return (
    <Box sx={{ mx: 2, mt: 1.25, flexShrink: 0 }}>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: `${ROW_GAP}px`,
          maxHeight: VISIBLE_ROWS * ROW_HEIGHT + (VISIBLE_ROWS - 1) * ROW_GAP,
          overflowY: 'auto',
        }}
      >
        {problems.map((p) => {
          const info = isInfoRow(p)
          return (
            <Box
              key={JSON.stringify(p)}
              role="alert"
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 1.25,
                minHeight: ROW_HEIGHT,
                flexShrink: 0,
                pl: 1.5,
                pr: 0.75,
                fontSize: 14,
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
              <Typography noWrap={true} sx={{ flex: 1, minWidth: 0, fontSize: 14 }}>
                {p.kind === 'drift' ? describeDrift(p.drift) : describe(p)}
              </Typography>
              {p.kind === 'drift' ? <DriftButtons drift={p.drift} /> : <FixButton problem={p} />}
            </Box>
          )
        })}
      </Box>
      {result?.unknown ? (
        <Typography sx={{ mt: 0.5, fontSize: 12, color: 'text.secondary' }}>
          {t`Some checks could not run without a connection, so more problems may show up later.`}
        </Typography>
      ) : null}
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', mt: 0.75 }}>
        <Button size="small" color="inherit" onClick={() => setTab('problems')}>
          {t`See all`}
        </Button>
      </Box>
    </Box>
  )
}
