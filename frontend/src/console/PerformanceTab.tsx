import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Collapse, Typography } from '@mui/material'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useState } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { PerformancePanel } from './PerformancePanel.tsx'
import { StartupPanel } from './StartupPanel.tsx'
import { usePerfQuery, useSmapiStartup } from './startupHooks.ts'
import { useSavedReports } from './usePerformancePanel.ts'

// The in-game reports are saved per profile, so a profile with none starts with the section closed.
function InGame({ game }: { game: string }) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const [saved] = useSavedReports(game, openId)
  const [choice, setChoice] = useState<boolean | null>(null)
  const open = choice ?? saved.length > 0
  return (
    <Box component="section">
      <ButtonBase
        aria-expanded={open}
        onClick={() => setChoice(!open)}
        sx={{ gap: 0.75, borderRadius: '4px', justifyContent: 'flex-start' }}
      >
        {open ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
        <Typography component="h3" sx={{ fontSize: 18, fontWeight: 600 }}>
          {t`In game`}
        </Typography>
      </ButtonBase>
      <Collapse in={open} unmountOnExit={true}>
        <PerformancePanel game={game} />
      </Collapse>
    </Box>
  )
}

// In game reads SMAPI's performance counters through its console, or a companion that measures in game.
export function PerformanceTab({ game }: { game: string }) {
  const smapi = useSmapiStartup()
  const perf = usePerfQuery()
  return (
    <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
      <StartupPanel game={game}>{smapi || perf ? <InGame game={game} /> : null}</StartupPanel>
    </Box>
  )
}
