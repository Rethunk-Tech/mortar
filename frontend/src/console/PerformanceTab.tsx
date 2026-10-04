import { useLingui } from '@lingui/react/macro'
import { Box, ToggleButton, ToggleButtonGroup } from '@mui/material'
import { useState } from 'react'
import { PerformancePanel } from './PerformancePanel.tsx'
import { StartupPanel } from './StartupPanel.tsx'

type View = 'startup' | 'inGame'

export function PerformanceTab({ game }: { game: string }) {
  const { t } = useLingui()
  const [view, setView] = useState<View>('startup')
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ px: 2, pt: 1.5 }}>
        <ToggleButtonGroup
          size="small"
          exclusive={true}
          value={view}
          onChange={(_, v: View | null) => {
            if (v) {
              setView(v)
            }
          }}
          aria-label={t`Performance view`}
        >
          <ToggleButton value="startup">{t`Startup`}</ToggleButton>
          <ToggleButton value="inGame">{t`In game`}</ToggleButton>
        </ToggleButtonGroup>
      </Box>
      {view === 'startup' ? <StartupPanel game={game} /> : <PerformancePanel game={game} />}
    </Box>
  )
}
