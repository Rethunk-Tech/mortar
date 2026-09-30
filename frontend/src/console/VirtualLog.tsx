import { Box } from '@mui/material'
import { useLayoutEffect, useRef, useState } from 'react'
import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'

// 13px monospace at line-height 1.75.
const ROW = 23
const OVERSCAN = 20
// A jumped-to row lands a third of the way down the view.
const JUMP_DIVISOR = 3

interface Look {
  color: string
  bar: string
  row: string
  text: string
}

const CLEAR = 'transparent'
const looks: Record<Level, Look> = {
  [Level.$zero]: { color: '', bar: CLEAR, row: CLEAR, text: '' },
  [Level.Trace]: {
    color: 'rgba(175,175,185,0.9)',
    bar: CLEAR,
    row: CLEAR,
    text: 'rgba(190,190,200,0.9)',
  },
  [Level.Debug]: {
    color: 'rgba(195,195,205,0.95)',
    bar: CLEAR,
    row: CLEAR,
    text: 'rgba(210,210,220,0.95)',
  },
  [Level.Info]: {
    color: 'rgba(235,235,240,0.95)',
    bar: CLEAR,
    row: CLEAR,
    text: 'rgba(235,235,240,0.95)',
  },
  [Level.Warn]: { color: '#F3B416', bar: '#F3B416', row: 'rgba(243,180,22,0.08)', text: '#f7d56a' },
  [Level.Error]: {
    color: '#ff9a90',
    bar: '#ff6b5f',
    row: 'rgba(255,107,95,0.10)',
    text: '#ffc4be',
  },
  [Level.Alert]: {
    color: '#c792ea',
    bar: '#c792ea',
    row: 'rgba(199,146,234,0.10)',
    text: '#e3c6f5',
  },
}

const cell = { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'pre' } as const

function Row({ entry, timestamps }: { entry: Entry; timestamps: boolean }) {
  const look = looks[entry.level]
  return (
    <Box
      sx={{
        height: ROW,
        display: 'grid',
        gridTemplateColumns: timestamps
          ? '4px 72px 56px 150px minmax(0, 1fr)'
          : '4px 56px 150px minmax(0, 1fr)',
        gap: '10px',
        alignItems: 'center',
        pr: 1.5,
        bgcolor: look.row,
      }}
    >
      <Box sx={{ alignSelf: 'stretch', bgcolor: look.bar }} />
      {timestamps ? (
        <Box sx={{ ...cell, color: 'rgba(175,175,185,0.9)' }}>{entry.cont ? '' : entry.time}</Box>
      ) : null}
      <Box sx={{ ...cell, color: look.color, fontWeight: 500 }}>
        {entry.cont ? '' : entry.level}
      </Box>
      <Box sx={{ ...cell, color: 'rgba(214,214,220,0.95)' }}>{entry.cont ? '' : entry.mod}</Box>
      <Box sx={{ ...cell, color: look.text }} title={entry.message}>
        {entry.message}
      </Box>
    </Box>
  )
}

// A fixed-height window over the rows: only the ones in view (plus a margin) are in the DOM.
export function VirtualLog({
  rows,
  timestamps,
  follow,
  onUnfollow,
  jump,
}: {
  rows: Entry[]
  timestamps: boolean
  follow: boolean
  onUnfollow: () => void
  // Each request carries a fresh n so jumping to the same row twice scrolls twice.
  jump: { index: number; n: number } | null
}) {
  const ref = useRef<HTMLDivElement>(null)
  const [top, setTop] = useState(0)
  const [height, setHeight] = useState(0)

  useLayoutEffect(() => {
    const el = ref.current
    if (!el) {
      return
    }
    const measure = new ResizeObserver(() => setHeight(el.clientHeight))
    measure.observe(el)
    return () => measure.disconnect()
  }, [])

  const count = rows.length
  useLayoutEffect(() => {
    const el = ref.current
    if (follow && el) {
      // The browser clamps to the real bottom.
      el.scrollTop = count * ROW
    }
  }, [follow, count])

  useLayoutEffect(() => {
    const el = ref.current
    if (jump && el) {
      el.scrollTop = Math.max(0, jump.index * ROW - el.clientHeight / JUMP_DIVISOR)
    }
  }, [jump])

  const start = Math.max(0, Math.floor(top / ROW) - OVERSCAN)
  const end = Math.min(rows.length, Math.ceil((top + height) / ROW) + OVERSCAN)
  return (
    <Box
      ref={ref}
      role="log"
      onScroll={(e) => {
        const el = e.currentTarget
        setTop(el.scrollTop)
        if (follow && el.scrollHeight - el.clientHeight - el.scrollTop > ROW) {
          onUnfollow()
        }
      }}
      sx={{ height: '100%', overflowY: 'auto', userSelect: 'text' }}
    >
      <Box sx={{ height: rows.length * ROW, position: 'relative' }}>
        <Box sx={{ transform: `translateY(${start * ROW}px)` }}>
          {rows.slice(start, end).map((entry) => (
            <Row key={entry.seq} entry={entry} timestamps={timestamps} />
          ))}
        </Box>
      </Box>
    </Box>
  )
}
