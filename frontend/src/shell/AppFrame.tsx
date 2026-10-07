import { Box } from '@mui/material'
import { Events } from '@wailsio/runtime'
import { type ReactNode, useEffect, useState } from 'react'
import { DropOverlay } from '../install/DropOverlay.tsx'
import { SweepDialog } from '../launch/SweepDialog.tsx'
import { QueueSheet } from '../queue/QueueSheet.tsx'
import { useSettings } from '../settings/store.ts'
import { OfflineBanner } from './OfflineBanner.tsx'
import { TitleBar } from './TitleBar.tsx'
import { win } from './win.ts'

const TINT = 'var(--mortar-tint)'
const WIDTH_STEP_PX = 320
const MAX_BACKDROP_PX = 2560

// The backdrop is asked for at the window's pixel width (rounded up to a step, capped), so a 4K wallpaper is not
// decoded in full behind a smaller window.
const backdropWidth = (): number =>
  Math.min(
    MAX_BACKDROP_PX,
    Math.ceil((window.innerWidth * window.devicePixelRatio) / WIDTH_STEP_PX) * WIDTH_STEP_PX,
  )

export function AppFrame({ children }: { children: ReactNode }) {
  const [maximised, setMaximised] = useState(false)
  const showBackdrop = useSettings((s) => s.background !== 'solid')
  const backdropImage = useSettings((s) => s.backgroundImage)
  const backdropMode = useSettings((s) => s.background)
  const [width] = useState(backdropWidth)
  const [frame, setFrame] = useState<HTMLElement | null>(null)
  useEffect(() => {
    const sync = () => win.reportMaximised(setMaximised)
    sync()
    return Events.On(Events.Types.Common.WindowDidResize, sync)
  }, [])
  return (
    <Box
      data-file-drop-target=""
      ref={setFrame}
      sx={{
        position: 'fixed',
        inset: 0,
        display: 'flex',
        flexDirection: 'column',
        // A hidden overflow is still a scroll container: focusing a child below the fold slid the whole window up.
        overflow: 'clip',
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
            src={`/backdrop?mode=${backdropMode}&w=${width}&v=${encodeURIComponent(backdropImage)}`}
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
      <OfflineBanner />
      <Box component="main" sx={{ flexGrow: 1, minHeight: 0 }}>
        {children}
      </Box>
      <QueueSheet />
      <DropOverlay target={frame} />
      <SweepDialog />
    </Box>
  )
}
