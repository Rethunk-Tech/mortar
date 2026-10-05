import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { type PointerEvent, type ReactNode, useRef, useState } from 'react'
import { useStoredState } from '../shell/useStoredState.ts'
import {
  clampPanelWidth,
  isWidth,
  KEY_STEP_PX,
  PANEL_DEFAULT_PX,
  PANEL_MIN_PX,
} from './panelWidth.ts'

// The details column, resized by dragging its left edge or with the arrow keys on the handle. The width is clamped
// to the Mods tab it sits in, so a narrow window never leaves the list without room.
export function ResizableAside({ label, children }: { label: string; children: ReactNode }) {
  const { t } = useLingui()
  const [stored, setStored] = useStoredState('mortar.modPanelWidth', PANEL_DEFAULT_PX, isWidth)
  const [drag, setDrag] = useState<number | null>(null)
  const ref = useRef<HTMLElement>(null)
  const available = () => ref.current?.parentElement?.clientWidth ?? Number.POSITIVE_INFINITY
  const width = clampPanelWidth(drag ?? stored, available())
  const onMove = (e: PointerEvent<HTMLElement>) => {
    if (drag !== null && ref.current) {
      setDrag(clampPanelWidth(ref.current.getBoundingClientRect().right - e.clientX, available()))
    }
  }
  const finish = () => {
    if (drag !== null) {
      setStored(drag)
      setDrag(null)
    }
  }
  return (
    <Box
      aria-label={label}
      component="aside"
      ref={ref}
      sx={{
        position: 'relative',
        width,
        overflowY: 'auto',
        bgcolor: 'var(--mortar-panel)',
        borderLeft: '1px solid var(--mortar-hairline)',
      }}
    >
      <Box
        role="separator"
        aria-orientation="vertical"
        aria-label={t`Resize details`}
        aria-valuenow={Math.round(width)}
        aria-valuemin={PANEL_MIN_PX}
        tabIndex={0}
        onPointerDown={(e) => {
          e.currentTarget.setPointerCapture(e.pointerId)
          setDrag(width)
        }}
        onPointerMove={onMove}
        onPointerUp={finish}
        onPointerCancel={finish}
        onKeyDown={(e) => {
          const step = { ArrowLeft: KEY_STEP_PX, ArrowRight: -KEY_STEP_PX }[e.key]
          if (step !== undefined) {
            e.preventDefault()
            setStored(clampPanelWidth(width + step, available()))
          }
        }}
        sx={{
          position: 'absolute',
          insetBlock: 0,
          left: 0,
          width: 6,
          cursor: 'col-resize',
          zIndex: 1,
          '&:hover, &:focus-visible': { bgcolor: 'primary.main' },
        }}
      />
      {children}
    </Box>
  )
}
