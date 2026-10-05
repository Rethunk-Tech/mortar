import { useLingui } from '@lingui/react/macro'
import { Box, Drawer } from '@mui/material'
import { type KeyboardEvent, useEffect, useRef } from 'react'
import { useSettings } from '../settings/store.ts'
import { Row } from './GameSelect.tsx'
import { useGameTiles } from './useGameTiles.ts'

// Game Select's tiles, shrunk to one row: 240 px is the tile's own minimum height, enough for its name, profile
// cards and 96 px source badges without wrapping.
const CARD_WIDTH_PX = 560
const CARD_HEIGHT_PX = 240
const NEWEST = '￿'

// moveFocus steps to the neighbouring card's button, so Left and Right walk the row and Enter opens the game.
function moveFocus(e: KeyboardEvent<HTMLElement>) {
  const step = { ArrowLeft: -1, ArrowRight: 1 }[e.key]
  if (step === undefined) {
    return
  }
  const covers = [...e.currentTarget.querySelectorAll<HTMLElement>('[data-game-cover]')]
  const at = covers.indexOf(document.activeElement as HTMLElement)
  covers[Math.min(Math.max(at + step, 0), covers.length - 1)]?.focus()
  e.preventDefault()
}

// A top drawer under the title bar listing the playable games, the game opened most recently first, then the rest
// by when they were last played. Choosing one opens it on its last profile; the route change closes the drawer.
export function GameSwitcher({
  current,
  open,
  onClose,
}: {
  current: string
  open: boolean
  onClose: () => void
}) {
  const { t } = useLingui()
  const { status, tileProps } = useGameTiles()
  const lastGame = useSettings((s) => s.lastGame)
  const played = useSettings((s) => s.lastPlayed)
  const row = useRef<HTMLDivElement>(null)
  const at = (id: string) => (id === lastGame ? NEWEST : (played?.[id]?.at ?? ''))
  const games = (status?.games ?? [])
    .filter((g) => g.available)
    .sort((a, b) => at(b.id).localeCompare(at(a.id)))
  const ready = games.length > 0
  useEffect(() => {
    if (open && ready) {
      requestAnimationFrame(() =>
        row.current?.querySelector<HTMLElement>('[aria-current="true"]')?.focus(),
      )
    }
  }, [open, ready])
  return (
    <Drawer
      anchor="top"
      open={open}
      onClose={onClose}
      sx={{ top: 'var(--title-bar)' }}
      slotProps={{
        paper: {
          role: 'dialog',
          'aria-label': t`Switch game`,
          sx: { top: 'var(--title-bar)', bgcolor: 'var(--mortar-panel-92)' },
        },
      }}
    >
      <Box
        ref={row}
        onKeyDown={moveFocus}
        sx={{ display: 'flex', overflowX: 'auto', overflowY: 'hidden' }}
      >
        {games.map((g) => (
          <Box
            key={g.id}
            sx={{ width: CARD_WIDTH_PX, height: CARD_HEIGHT_PX, flexShrink: 0, display: 'grid' }}
          >
            <Row {...tileProps(g)} selected={g.id === current} />
          </Box>
        ))}
      </Box>
    </Drawer>
  )
}
