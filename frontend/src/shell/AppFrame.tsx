import { Box } from '@mui/material'
import { Events } from '@wailsio/runtime'
import { type ReactNode, useEffect, useState } from 'react'
import { DropOverlay } from '../install/DropOverlay.tsx'
import { dropTargetProps } from '../install/dropTarget.ts'
import { useSettings } from '../settings/store.ts'
import { TitleBar } from './TitleBar.tsx'
import { win } from './win.ts'

const TINT = 'var(--mortar-tint)'

export function AppFrame({ children }: { children: ReactNode }) {
  const [maximised, setMaximised] = useState(false)
  const showBackdrop = useSettings((s) => s.background !== 'solid')
  const backdropImage = useSettings((s) => s.backgroundImage)
  const backdropMode = useSettings((s) => s.background)
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
        border: maximised ? 0 : '1px solid var(--mortar-hairline-12)',
        borderRadius: maximised ? 0 : '10px',
        userSelect: 'none',
      }}
    >
      {showBackdrop ? (
        <>
          <Box
            component="img"
            alt=""
            draggable={false}
            src={`/backdrop?mode=${backdropMode}&v=${encodeURIComponent(backdropImage)}`}
            sx={{
              position: 'absolute',
              inset: 0,
              zIndex: -1,
              width: '100%',
              height: '100%',
              objectFit: 'cover',
              pointerEvents: 'none',
            }}
          />
          <Box
            sx={{
              position: 'absolute',
              inset: 0,
              zIndex: -1,
              bgcolor: TINT,
              pointerEvents: 'none',
            }}
          />
        </>
      ) : null}
      <TitleBar maximised={maximised} />
      <Box component="main" sx={{ flexGrow: 1, minHeight: 0 }}>
        {children}
      </Box>
      <DropOverlay target={frame} />
    </Box>
  )
}
