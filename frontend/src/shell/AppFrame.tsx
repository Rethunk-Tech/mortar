import { Box } from '@mui/material'
import { Events } from '@wailsio/runtime'
import { type ReactNode, useEffect, useState } from 'react'
import { TitleBar } from './TitleBar.tsx'
import { win } from './win.ts'

export function AppFrame({ children }: { children: ReactNode }) {
  const [maximised, setMaximised] = useState(false)
  useEffect(() => {
    const sync = () => win.reportMaximised(setMaximised)
    sync()
    return Events.On(Events.Types.Common.WindowDidResize, sync)
  }, [])
  return (
    <Box
      sx={{
        position: 'fixed',
        inset: 0,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        boxSizing: 'border-box',
        border: maximised ? 0 : '1px solid rgba(255,255,255,0.12)',
        borderRadius: maximised ? 0 : '10px',
        userSelect: 'none',
      }}
    >
      <TitleBar maximised={maximised} />
      <Box component="main" sx={{ flexGrow: 1, minHeight: 0 }}>
        {children}
      </Box>
    </Box>
  )
}
