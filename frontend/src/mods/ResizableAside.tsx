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

// The details panel, laid over the right of the list so opening it never changes the list's width (and so never
// re-lays or redraws its rows); resized by dragging its left edge or with the arrow keys on the handle, clamped to the
// tab it sits in.
export function ResizableAside({ label, children }: { label: string; children: ReactNode }) {
  const { t } = useLingui()
  const [stored, setStored] = useStoredState('mortar.modPanelWidth', PANEL_DEFAULT_PX, isWidth)
  const [drag, setDrag] = useState<number | null>(null)
  const ref = useRef<HTMLElement>(null)
  const available = () => ref.current?.parentElement?.clientWidth ?? Number.POSITIVE_INFINITY
  const width = clampPanelWidth(drag ?? stored, available())
  const widest = clampPanelWidth(Number.POSITIVE_INFINITY, available())
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
        position: 'absolute',
        insetBlock: 0,
        right: 0,
        zIndex: 2,
        width,
        overflowY: 'auto',
        bgcolor: 'var(--mortar-panel-solid)',
        borderLeft: '1px solid var(--mortar-hairline)',
        boxShadow: '-6px 0 18px var(--mortar-overlay-45)',
      }}
    >
      <Box
        role="separator"
        aria-orientation="vertical"
        aria-label={t`Resize details`}
        aria-valuenow={Math.round(width)}
        aria-valuemin={PANEL_MIN_PX}
        aria-valuemax={Number.isFinite(widest) ? Math.round(widest) : undefined}
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
