import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { arrowFocus } from '@rethunk/dom-input-nav'
import type { ReactNode } from 'react'
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

// The first step of the Share and Import dialogs: one tile per way to send or bring in a profile.
export function TileGrid<Id extends string>({
  label,
  tiles,
  lastUsed,
  onChoose,
  selected,
  padded = true,
}: {
  label: string
  tiles: Tile<Id>[]
  lastUsed: Id | null
  onChoose: (id: Id) => void
  // The tile a wizard step already holds, outlined and focused.
  selected?: Id
  // False inside a body that already pads its content.
  padded?: boolean
}) {
  const { t } = useLingui()
  return (
    <Box
      role="group"
      aria-label={label}
      onKeyDown={(event) => arrowFocus(event, 'button:not(:disabled)')}
      sx={{
        display: 'grid',
        gridTemplateColumns: `repeat(${gridColumns(tiles.length)}, minmax(0, 1fr))`,
        gap: space.gap,
        p: padded ? '24px' : 0,
        alignContent: 'start',
        overflowY: 'auto',
      }}
    >
      {tiles.map((tile) => (
        <Box key={tile.id} sx={{ display: 'flex' }}>
          <ButtonBase
            disabled={tile.disabled !== undefined}
            autoFocus={tile.id === (selected ?? lastUsed)}
            aria-pressed={selected === undefined ? undefined : tile.id === selected}
            onClick={() => onChoose(tile.id)}
            sx={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'flex-start',
              justifyContent: 'flex-start',
              position: 'relative',
              gap: space.gap,
              p: space.pad,
              textAlign: 'left',
              borderRadius: '10px',
              border: '1px solid',
              borderColor: tile.id === selected ? 'primary.main' : 'var(--mortar-hairline-12)',
              bgcolor: 'var(--mortar-card-hover)',
              opacity: tile.disabled === undefined ? 1 : DISABLED_OPACITY,
              '&:hover, &:focus-visible': { borderColor: 'primary.main' },
            }}
          >
            {tile.icon}
            <Box sx={{ minWidth: 0 }}>
              <Typography sx={{ fontWeight: 600, fontSize: 'lead.fontSize' }}>
                {tile.name}
              </Typography>
              <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
                {tile.hint}
              </Typography>
              {tile.disabled ? (
                <Typography sx={{ fontSize: 'small.fontSize', color: 'text.primary', mt: 0.5 }}>
                  {tile.disabled}
                </Typography>
              ) : null}
            </Box>
            {/* In the corner, so marking a tile never changes its height. */}
            {tile.id === lastUsed ? (
              <Typography
                sx={{
                  position: 'absolute',
                  top: space.pad,
                  right: space.pad,
                  fontSize: 'tiny.fontSize',
                  color: 'var(--mortar-accent-ink)',
                }}
              >
                {t`Last used`}
              </Typography>
            ) : null}
          </ButtonBase>
        </Box>
      ))}
    </Box>
  )
}
