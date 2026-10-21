import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import type { KeyboardEvent, ReactNode } from 'react'
import { space } from '../theme/density.ts'
import { gridColumns } from './methods.ts'

const DISABLED_OPACITY = 0.45

export interface Tile<Id extends string> {
  id: Id
  icon: ReactNode
  name: string
  hint: string
  // Why the tile cannot be used; it stays in place so the grid does not shift.
  disabled?: string
}

// Arrow keys move between the enabled tiles; the grid's own column count decides what up and down mean.
function moveFocus(event: KeyboardEvent<HTMLElement>) {
  const grid = event.currentTarget
  const tiles = [...grid.querySelectorAll<HTMLElement>('button:not(:disabled)')]
  const at = tiles.indexOf(document.activeElement as HTMLElement)
  const columns = getComputedStyle(grid).gridTemplateColumns.split(' ').length
  const step: Record<string, number> = {
    ArrowRight: 1,
    ArrowLeft: -1,
    ArrowDown: columns,
    ArrowUp: -columns,
  }
  const delta = step[event.key]
  if (delta === undefined || at < 0) {
    return
  }
  const next = tiles[at + delta]
  if (next) {
    event.preventDefault()
    next.focus()
  }
}

// The first step of the Share and Import dialogs: one tile per way to send or bring in a profile.
export function TileGrid<Id extends string>({
  label,
  tiles,
  lastUsed,
  onChoose,
}: {
  label: string
  tiles: Tile<Id>[]
  lastUsed: Id | null
  onChoose: (id: Id) => void
}) {
  const { t } = useLingui()
  return (
    <Box
      role="group"
      aria-label={label}
      onKeyDown={moveFocus}
      sx={{
        display: 'grid',
        gridTemplateColumns: `repeat(${gridColumns(tiles.length)}, minmax(0, 1fr))`,
        gap: space.gap,
        p: '24px',
        alignContent: 'start',
        overflowY: 'auto',
      }}
    >
      {tiles.map((tile) => (
        <Box key={tile.id} sx={{ display: 'flex' }}>
          <ButtonBase
            disabled={tile.disabled !== undefined}
            autoFocus={tile.id === lastUsed}
            onClick={() => onChoose(tile.id)}
            sx={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'flex-start',
              gap: space.gap,
              p: space.pad,
              textAlign: 'left',
              borderRadius: '10px',
              border: '1px solid var(--mortar-hairline-12)',
              bgcolor: 'var(--mortar-card-hover)',
              opacity: tile.disabled === undefined ? 1 : DISABLED_OPACITY,
              '&:hover, &:focus-visible': { borderColor: 'primary.main' },
            }}
          >
            {tile.icon}
            <Box sx={{ minWidth: 0 }}>
              <Typography sx={{ fontWeight: 600, fontSize: 15 }}>{tile.name}</Typography>
              <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>{tile.hint}</Typography>
              {tile.disabled ? (
                <Typography sx={{ fontSize: 12, color: 'text.primary', mt: 0.5 }}>
                  {tile.disabled}
                </Typography>
              ) : null}
              {tile.id === lastUsed ? (
                <Typography sx={{ fontSize: 11, color: 'var(--mortar-accent-ink)', mt: 0.5 }}>
                  {t`Last used`}
                </Typography>
              ) : null}
            </Box>
          </ButtonBase>
        </Box>
      ))}
    </Box>
  )
}
