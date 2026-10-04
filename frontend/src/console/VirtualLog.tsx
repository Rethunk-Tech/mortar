import { useLingui } from '@lingui/react/macro'
import { Box, Link } from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { type ReactNode, useLayoutEffect, useMemo, useRef, useState } from 'react'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import {
  type ConsoleLink,
  type ConsoleLinkRoots,
  type InstalledMod,
  linksForModColumn,
  linksInText,
} from './consoleLinks.ts'
import { levelChrome } from './levelPalette.ts'
import type { UpdateNote } from './smapiUpdateNotes.ts'
import { UpdateNoteTag } from './UpdateNoteTag.tsx'

// 13px monospace at line-height 1.75.
const ROW = 23
const OVERSCAN = 20
// A jumped-to row lands a third of the way down the view.
const JUMP_DIVISOR = 3

const cell = { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'pre' } as const

const linkSx = { font: 'inherit', verticalAlign: 'baseline', textUnderlineOffset: '2px' } as const

function Linked({
  text,
  links,
  onMod,
  onPath,
}: {
  text: string
  links: ConsoleLink[]
  onMod: (uniqueID: string) => void
  onPath: (path: string) => void
}) {
  if (links.length === 0) {
    return text
  }
  const parts: ReactNode[] = []
  let at = 0
  for (const link of links) {
    if (link.start > at) {
      parts.push(text.slice(at, link.start))
    }
    const label = text.slice(link.start, link.end)
    parts.push(
      <Link
        key={`${link.kind}-${link.start}`}
        component="button"
        type="button"
        color="inherit"
        underline="always"
        onClick={() => {
          if (link.kind === 'mod') {
            onMod(link.uniqueID)
          } else {
            onPath(link.path)
          }
        }}
        sx={linkSx}
      >
        {label}
      </Link>,
    )
    at = link.end
  }
  if (at < text.length) {
    parts.push(text.slice(at))
  }
  return parts
}

function Row({
  entry,
  timestamps,
  mods,
  roots,
  onMod,
  onPath,
  note,
}: {
  entry: Entry
  note: UpdateNote | undefined
  timestamps: boolean
  mods: readonly InstalledMod[]
  roots: ConsoleLinkRoots
  onMod: (uniqueID: string) => void
  onPath: (path: string) => void
}) {
  const theme = useTheme()
  const look = levelChrome(theme, entry.level)
  const modLinks = useMemo(
    () => (entry.cont ? [] : linksForModColumn(entry.mod, mods)),
    [entry.cont, entry.mod, mods],
  )
  const messageLinks = useMemo(
    () => linksInText(entry.message, mods, roots),
    [entry.message, mods, roots],
  )
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
        <Box sx={{ ...cell, color: 'text.secondary' }}>{entry.cont ? '' : entry.time}</Box>
      ) : null}
      <Box sx={{ ...cell, color: look.color, fontWeight: 500 }}>
        {entry.cont ? '' : entry.level}
      </Box>
      <Box sx={{ ...cell, color: 'text.secondary' }}>
        {entry.cont ? (
          ''
        ) : (
          <Linked text={entry.mod} links={modLinks} onMod={onMod} onPath={onPath} />
        )}
      </Box>
      <Box
        sx={{ display: 'flex', alignItems: 'center', minWidth: 0, color: look.text }}
        title={entry.message}
      >
        <Box sx={cell}>
          <Linked text={entry.message} links={messageLinks} onMod={onMod} onPath={onPath} />
        </Box>
        {note ? <UpdateNoteTag note={note} /> : null}
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
  mods,
  roots,
  onMod,
  onPath,
  notes,
}: {
  rows: Entry[]
  notes: ReadonlyMap<number, UpdateNote>
  timestamps: boolean
  follow: boolean
  onUnfollow: () => void
  // Each request carries a fresh n so jumping to the same row twice scrolls twice.
  jump: { index: number; n: number } | null
  mods: readonly InstalledMod[]
  roots: ConsoleLinkRoots
  onMod: (uniqueID: string) => void
  onPath: (path: string) => void
}) {
  const { t } = useLingui()
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
      aria-label={t`Game log`}
      // Scrolling mounts and unmounts rows, which a live log would read out as new output.
      aria-live="off"
      tabIndex={0}
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
            <Row
              key={entry.seq}
              entry={entry}
              note={notes.get(entry.seq)}
              timestamps={timestamps}
              mods={mods}
              roots={roots}
              onMod={onMod}
              onPath={onPath}
            />
          ))}
        </Box>
      </Box>
    </Box>
  )
}
