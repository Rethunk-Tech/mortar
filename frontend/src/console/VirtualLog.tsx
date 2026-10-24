import { useLingui } from '@lingui/react/macro'
import { Box, Link } from '@mui/material'
import { useTheme } from '@mui/material/styles'
import {
  memo,
  type ReactNode,
  useCallback,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { space } from '../theme/density.ts'
import {
  type ConsoleLink,
  type ConsoleLinkRoots,
  type InstalledMod,
  linksForModColumn,
  linksInText,
} from './consoleLinks.ts'
import { repeatCount } from './filter.ts'
import { LineMenu } from './LineMenu.tsx'
import { levelChrome } from './levelPalette.ts'
import { useLineText } from './lineText.ts'
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
  onMod: (id: string) => void
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
            onMod(link.id)
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

// Memoised: scrolling moves the window over the rows, and a row already in view need not draw again.
const Row = memo(function RowView({
  entry,
  timestamps,
  mods,
  roots,
  onMod,
  onPath,
  onMenu,
  note,
}: {
  entry: Entry
  note: UpdateNote | undefined
  onMenu: (seq: number, top: number, left: number) => void
  timestamps: boolean
  mods: readonly InstalledMod[]
  roots: ConsoleLinkRoots
  onMod: (id: string) => void
  onPath: (path: string) => void
}) {
  const { t } = useLingui()
  const theme = useTheme()
  const text = useLineText()(entry)
  const look = levelChrome(theme, entry.level)
  const repeat = repeatCount(entry)
  const modLinks = useMemo(
    () => (entry.cont ? [] : linksForModColumn(entry.mod, mods)),
    [entry.cont, entry.mod, mods],
  )
  const messageLinks = useMemo(() => linksInText(text, mods, roots), [text, mods, roots])
  return (
    <Box
      onContextMenu={(e) => {
        e.preventDefault()
        onMenu(entry.seq, e.clientY, e.clientX)
      }}
      sx={{
        height: ROW,
        display: 'grid',
        gridTemplateColumns: timestamps
          ? '4px 72px 56px 150px minmax(0, 1fr)'
          : '4px 56px 150px minmax(0, 1fr)',
        gap: '10px',
        alignItems: 'center',
        pr: space.pad,
        bgcolor: look.row,
      }}
    >
      <Box sx={{ alignSelf: 'stretch', bgcolor: look.bar }} />
      {timestamps ? (
        <Box sx={{ ...cell, color: 'text.secondary' }} title={entry.cont ? undefined : entry.time}>
          {entry.cont ? '' : entry.time.replace(FRACTION, '$1')}
        </Box>
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
        title={text}
      >
        <Box sx={cell}>
          <Linked text={text} links={messageLinks} onMod={onMod} onPath={onPath} />
        </Box>
        {repeat > 1 ? (
          <Box
            component="span"
            title={t`Written ${repeat} times in a row`}
            sx={{
              ml: 1,
              px: 0.75,
              flexShrink: 0,
              borderRadius: '10px',
              fontSize: 'small.fontSize',
              fontVariantNumeric: 'tabular-nums',
              bgcolor: 'var(--mortar-hairline-16)',
              color: 'text.secondary',
            }}
          >
            {`×${repeat}`}
          </Box>
        ) : null}
        {note ? <UpdateNoteTag note={note} /> : null}
      </Box>
    </Box>
  )
})

// A fixed-height window over the rows: only the ones in view (plus a margin) are in the DOM.
// A BepInEx logger mod's clock carries a fraction the 72px column has no room for; the title keeps it.
const FRACTION = /^(\d\d:\d\d:\d\d)\.\d+$/

// The message the row at index belongs to: its header line and the continuation lines after it.
function groupAt(rows: Entry[], index: number): Entry[] {
  let start = index
  while (start > 0 && rows[start]?.cont) {
    start -= 1
  }
  let end = start + 1
  while (rows[end]?.cont) {
    end += 1
  }
  return rows.slice(start, end)
}

export function VirtualLog({
  game,
  rows,
  timestamps,
  follow,
  jump,
  mods,
  roots,
  onMod,
  onPath,
  notes,
}: {
  game: string
  rows: Entry[]
  notes: ReadonlyMap<number, UpdateNote>
  timestamps: boolean
  // New lines scroll the log down while it is at the bottom; scrolled up, it keeps its place.
  follow: boolean
  // Each request carries a fresh n so jumping to the same row twice scrolls twice.
  jump: { index: number; n: number } | null
  mods: readonly InstalledMod[]
  roots: ConsoleLinkRoots
  onMod: (id: string) => void
  onPath: (path: string) => void
}) {
  const { t } = useLingui()
  const ref = useRef<HTMLDivElement>(null)
  const [top, setTop] = useState(0)
  const [height, setHeight] = useState(0)
  const [menu, setMenu] = useState<{ seq: number; top: number; left: number } | null>(null)
  const onMenu = useCallback(
    (seq: number, y: number, x: number) => setMenu({ seq, top: y, left: x }),
    [],
  )
  const menuIndex = menu ? rows.findIndex((r) => r.seq === menu.seq) : -1

  useLayoutEffect(() => {
    const el = ref.current
    if (!el) {
      return
    }
    const measure = new ResizeObserver(() => setHeight(el.clientHeight))
    measure.observe(el)
    return () => measure.disconnect()
  }, [])

  // Whether the view sits at the bottom, kept from each scroll as it happens: lines arriving between a scroll and the
  // next render must not pull the view back down.
  const pinned = useRef(true)
  useLayoutEffect(() => {
    if (follow) {
      pinned.current = true
    }
  }, [follow])
  const count = rows.length
  useLayoutEffect(() => {
    const el = ref.current
    if (follow && pinned.current && el) {
      // The browser clamps to the real bottom.
      el.scrollTop = count * ROW
    }
  }, [follow, count])

  useLayoutEffect(() => {
    const el = ref.current
    if (jump && el) {
      pinned.current = false
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
        pinned.current = el.scrollHeight - el.clientHeight - el.scrollTop < ROW / 2
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
              onMenu={onMenu}
            />
          ))}
        </Box>
      </Box>
      <LineMenu
        game={game}
        at={menu && menuIndex >= 0 ? { top: menu.top, left: menu.left } : null}
        group={menuIndex >= 0 ? groupAt(rows, menuIndex) : []}
        onClose={() => setMenu(null)}
      />
    </Box>
  )
}
