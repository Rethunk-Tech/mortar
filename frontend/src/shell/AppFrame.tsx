import { Box } from '@mui/material'
import { Events } from '@wailsio/runtime'
import { type ReactNode, useEffect, useState } from 'react'
import { DropOverlay, dropTargetProps } from '../install/DropOverlay.tsx'
import { TitleBar } from './TitleBar.tsx'
import { win } from './win.ts'

export function AppFrame({ children }: { children: ReactNode }) {
  const [maximised, setMaximised] = useState(false)
  const [frame, setFrame] = useState<HTMLElement | null>(null)
  useEffect(() => {
    const sync = () => win.reportMaximised(setMaximised)
    sync()
    return Events.On(Events.Types.Common.WindowDidResize, sync)
  }, [])
  return (
    <Box
      {...dropTargetProps}
      ref={setFrame}
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
      <DropOverlay target={frame} />
    </Box>
  )
}
