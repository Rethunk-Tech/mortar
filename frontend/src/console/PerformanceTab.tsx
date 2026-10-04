import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { useState } from 'react'
import { TabPills } from '../share/TabPills.tsx'
import { PerformancePanel } from './PerformancePanel.tsx'
import { StartupPanel } from './StartupPanel.tsx'

type View = 'startup' | 'inGame'

export function PerformanceTab({ game }: { game: string }) {
  const { t } = useLingui()
  const [view, setView] = useState<View>('startup')
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ px: 2, pt: 1.5, display: 'flex' }}>
        <TabPills
          value={view}
          onChange={setView}
          label={t`Performance view`}
          options={[
            { value: 'startup', label: t`Startup` },
            { value: 'inGame', label: t`In game` },
          ]}
        />
      </Box>
      {view === 'startup' ? <StartupPanel game={game} /> : <PerformancePanel game={game} />}
    </Box>
  )
}
