import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Tooltip, Typography } from '@mui/material'
import { List, Wifi } from 'lucide-react'
import { type KeyboardEvent, type ReactNode, useRef } from 'react'
import { Logo } from '../brand/Logo.tsx'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { type Destination, type DestinationEntry, gridColumns } from './methods.ts'

const ICON = 32
const DISABLED_OPACITY = 0.45
const NEXUS = 'nexus'
const THUNDERSTORE = 'thunderstore'

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

export function DestinationGrid({
  entries,
  lastUsed,
  onChoose,
}: {
  entries: DestinationEntry[]
  lastUsed: Destination | null
  onChoose: (id: Destination) => void
}) {
  const { t } = useLingui()
  const gridRef = useRef<HTMLDivElement>(null)
  const tiles: Record<Destination, { icon: ReactNode; name: string; hint: string }> = {
    mortar: {
      icon: <Logo size={ICON} />,
      name: t`Mortar`,
      hint: t`A link or .mortar file`,
    },
    nexus: {
      icon: <SourceLogo id={NEXUS} size={ICON} />,
      name: t`Nexus Mods`,
      hint: t`A collection draft`,
    },
    thunderstore: {
      icon: <SourceLogo id={THUNDERSTORE} size={ICON} />,
      name: t`Thunderstore`,
      hint: t`An r2modman code or modpack`,
    },
    nearby: {
      icon: <Wifi size={ICON} />,
      name: t`Nearby computer`,
      hint: t`Send over your network`,
    },
    list: {
      icon: <List size={ICON} />,
      name: t`Mod list`,
      hint: t`Text for Discord or a forum`,
    },
  }
  return (
    <Box
      ref={gridRef}
      role="group"
      aria-label={t`Where to share`}
      onKeyDown={moveFocus}
      sx={{
        display: 'grid',
        gridTemplateColumns: `repeat(${gridColumns(entries.length)}, minmax(0, 1fr))`,
        gap: 1.5,
        p: '24px',
        alignContent: 'start',
        overflowY: 'auto',
      }}
    >
      {entries.map((entry) => {
        const tile = tiles[entry.id]
        return (
          <Tooltip
            key={entry.id}
            title={entry.disabled ? t`No mods to share.` : ''}
            disableInteractive={true}
          >
            <Box component="span" sx={{ display: 'flex' }}>
              <ButtonBase
                disabled={entry.disabled}
                autoFocus={entry.id === lastUsed}
                onClick={() => onChoose(entry.id)}
                sx={{
                  flex: 1,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'flex-start',
                  gap: 1,
                  p: 2,
                  textAlign: 'left',
                  borderRadius: '10px',
                  border: '1px solid var(--mortar-hairline-12)',
                  bgcolor: 'var(--mortar-card-hover)',
                  opacity: entry.disabled ? DISABLED_OPACITY : 1,
                  '&:hover, &:focus-visible': { borderColor: 'primary.main' },
                }}
              >
                {tile.icon}
                <Box sx={{ minWidth: 0 }}>
                  <Typography sx={{ fontWeight: 600, fontSize: 15 }}>{tile.name}</Typography>
                  <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
                    {tile.hint}
                  </Typography>
                  {entry.id === lastUsed ? (
                    <Typography sx={{ fontSize: 11, color: 'primary.main', mt: 0.5 }}>
                      {t`Last used`}
                    </Typography>
                  ) : null}
                </Box>
              </ButtonBase>
            </Box>
          </Tooltip>
        )
      })}
    </Box>
  )
}
