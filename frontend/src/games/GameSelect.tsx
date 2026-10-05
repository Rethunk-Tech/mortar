import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link, Typography } from '@mui/material'
import { Play } from 'lucide-react'
import type { MouseEvent } from 'react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Played } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import {
  SetLastGame,
  SetLastProfile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { gameSetupNeeded } from '../firstrun/needed.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useNow } from '../i18n/useNow.ts'
import { absoluteWhen } from '../i18n/when.ts'
import { playDirect } from '../launch/directPref.ts'
import { useLaunch } from '../launch/store.ts'
import { type GameId, isGameId, openSettings, useNav } from '../nav/store.ts'
import { CoverButton } from '../shell/CoverButton.tsx'
import { LoadErrorRow, LoadingRow } from '../shell/LoadingRow.tsx'
import { reportError } from '../toasts/report.ts'
import { gameArt } from './art.ts'
import { ProfileCards } from './ProfileCards.tsx'
import { formatPlaytime } from './playtime.ts'
import { storeName } from './storeName.ts'
import { ordered, useGameTiles } from './useGameTiles.ts'

type Game = GameInfo

const LONG_NAME = 8
// Tiles share the window in a grid that grows with the catalog: two games sit side by side, more wrap into rows.
const TILE_MIN_PX = 240
const TILE_MIN_WIDTH_PX = 560
// A compact tile: 96 px of badges between 24 px paddings.
const TILE_COMPACT_PX = 144
const SMALL_FONT = 13
const NORMAL_FONT = 15
const HOVER_MS = 200
const HOVER_EASE = 'cubic-bezier(0.2, 0.8, 0.2, 1)'
const shadow = '0 1px 2px var(--mortar-overlay-90), 0 0 18px var(--mortar-overlay-85)'

function fail(title: string, err: unknown) {
  reportError(title)(err)
}

// The row opens its game from a button laid under the content, so the Play button and profile cards on top are
// separate controls rather than controls nested inside another.
const aboveOpen = {
  position: 'relative',
  pointerEvents: 'none',
  '& button, & [role="button"], & a': { pointerEvents: 'auto' },
} as const

function Art({ src, openable }: { src: string; openable: boolean }) {
  return (
    <>
      <Box
        component="img"
        data-art=""
        src={src}
        alt=""
        sx={{
          // Slightly oversized so the parallax shift on hover never shows an edge.
          transform: 'scale(1.06)',
          position: 'absolute',
          inset: 0,
          width: '100%',
          height: '100%',
          objectFit: 'cover',
          opacity: 0.72,
          filter: openable ? 'none' : 'saturate(0.6)',
        }}
      />
      <Box
        sx={{
          position: 'absolute',
          inset: 0,
          bgcolor: openable ? 'var(--mortar-overlay-28)' : 'var(--mortar-overlay-55)',
        }}
      />
    </>
  )
}

function SourceBadges({ sources }: { sources: string[] }) {
  return (
    <>
      {sources.map((id) => {
        const name = sourceLabel(id)
        return (
          <Box
            key={id}
            sx={{
              width: 96,
              height: 96,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '8px',
              bgcolor: 'var(--mortar-game-dim)',
              fontSize: name.length > LONG_NAME ? SMALL_FONT : NORMAL_FONT,
              fontWeight: 700,
              color: '#fff',
            }}
          >
            <SourceLogo id={id} size={40} />
            {name}
          </Box>
        )
      })}
    </>
  )
}

export function Row({
  game,
  openable,
  note,
  loader,
  lastPlayedName,
  lastPlayedAt,
  lastPlayedId,
  playtimeMs,
  cards,
  selected = false,
  compact = false,
}: {
  game: Game
  openable: boolean
  note: string
  loader: string
  lastPlayedName: string
  lastPlayedAt: string
  lastPlayedId: string
  playtimeMs: number
  cards: { gameId: GameId; profiles: Profile[]; lastPlayed: Played | undefined } | undefined
  // The game already open, marked in the switcher.
  selected?: boolean
  // The switcher's card: the name, loader line, Play and sources, without the profile cards or count.
  compact?: boolean
}) {
  const { t, i18n } = useLingui()
  const start = useLaunch((s) => s.start)
  useNow()
  const ago = formatWhen(lastPlayedAt)
  let lastLine = ''
  if (lastPlayedName && ago) {
    lastLine = t`last played ${lastPlayedName} · ${ago}`
  } else if (lastPlayedName) {
    lastLine = t`last played ${lastPlayedName}`
  }
  const playtime = formatPlaytime(playtimeMs, i18n.locale)
  if (playtime) {
    const total = t`${playtime} played`
    lastLine = lastLine ? `${lastLine} · ${total}` : total
  }
  const open = () => {
    if (!isGameId(game.id)) {
      return
    }
    const { id } = game
    SetLastGame(id).catch((err: unknown) => fail(t`Could not save the last game`, err))
    gameSetupNeeded(game)
      .then((needed) =>
        needed ? useNav.getState().openGameSetup(id) : useNav.getState().openGame(id),
      )
      .catch((err: unknown) => fail(t`Could not read your games`, err))
  }
  const playLast = (ev: MouseEvent) => {
    ev.stopPropagation()
    if (!lastPlayedId) {
      return
    }
    SetLastGame(game.id).catch((err: unknown) => fail(t`Could not save the last game`, err))
    SetLastProfile(game.id, lastPlayedId).catch((err: unknown) =>
      fail(t`Could not save the open profile`, err),
    )
    useNav.getState().openGame(game.id)
    start(game.id, lastPlayedId, playDirect()).then(() => undefined)
  }
  const named = game.store ? storeName(game.store) : null
  const store = named ? t(named) : ''
  let loaderLine = loader
  if (store && lastLine) {
    loaderLine = t`${loader} | ${store} · ${lastLine}`
  } else if (store) {
    loaderLine = t`${loader} | ${store}`
  } else if (lastLine) {
    loaderLine = t`${loader} · ${lastLine}`
  }
  const content = (
    <>
      {gameArt(game) ? <Art src={gameArt(game)} openable={openable} /> : null}
      {openable ? (
        <CoverButton
          data-game-cover=""
          aria-current={selected ? 'true' : undefined}
          onClick={open}
          aria-label={t`Open ${game.name}`}
        />
      ) : null}
      <Box
        sx={{
          ...aboveOpen,
          minWidth: 0,
          overflow: 'hidden',
          textShadow: shadow,
          textAlign: 'left',
          color: '#fff',
        }}
      >
        <Typography sx={{ fontSize: 34, fontWeight: 600, lineHeight: 1.2 }}>{game.name}</Typography>
        <Typography
          title={lastPlayedAt ? absoluteWhen(lastPlayedAt, i18n.locale) || undefined : undefined}
          sx={{ fontSize: 17 }}
        >
          {loaderLine}
        </Typography>
        {compact ? null : (
          <Typography sx={{ mt: '6px', fontSize: 16, fontWeight: 600 }}>{note}</Typography>
        )}
        {cards && !compact ? (
          <ProfileCards
            gameId={cards.gameId}
            gameName={game.name}
            profiles={cards.profiles}
            lastPlayed={cards.lastPlayed}
          />
        ) : null}
      </Box>
      <Box sx={{ ...aboveOpen, display: 'flex', alignItems: 'center', gap: '14px' }}>
        {openable && lastPlayedId ? (
          <Button
            type="button"
            variant="contained"
            size="large"
            aria-label={t`Play ${lastPlayedName}`}
            startIcon={<Play size={22} fill="currentColor" />}
            onClick={playLast}
            sx={{
              flexShrink: 0,
              height: 96,
              minWidth: 120,
              px: 3,
              borderRadius: 0,
              fontSize: 17,
              fontWeight: 700,
              '& .MuiButton-startIcon': { mr: '10px' },
            }}
          >
            {t`Play`}
          </Button>
        ) : null}
        <SourceBadges sources={game.sources ?? []} />
      </Box>
    </>
  )
  const sx = {
    position: 'relative',
    minHeight: compact ? TILE_COMPACT_PX : TILE_MIN_PX,
    display: 'flex',
    flexWrap: 'wrap',
    alignItems: 'center',
    alignContent: 'center',
    justifyContent: 'space-between',
    gap: '16px',
    px: '48px',
    py: '24px',
    overflow: 'hidden',
    bgcolor: gameArt(game) ? 'transparent' : 'background.paper',
    borderLeft: '4px solid',
    borderColor: openable ? 'primary.main' : 'transparent',
    borderTop: '1px solid rgba(0,0,0,0.8)',
    fontFamily: 'inherit',
    ...(selected
      ? { outline: '2px solid', outlineColor: 'primary.main', outlineOffset: '-2px' }
      : {}),
  } as const
  return (
    <Box data-tile="" sx={sx}>
      {content}
    </Box>
  )
}

// Hover focus on the grid: the hovered tile grows, the rest shrink and dim, and the art drifts opposite ways. Only
// transform and filter change, so the grid never reflows (animating flex-grow or width jumped in WebKitGTK).
const hoverFocus = {
  '@media (hover: hover) and (prefers-reduced-motion: no-preference)': {
    '& [data-tile]': {
      willChange: 'transform, filter',
      transition: `transform ${HOVER_MS}ms ${HOVER_EASE}, filter ${HOVER_MS}ms ${HOVER_EASE}`,
    },
    '& [data-art]': { transition: `transform ${HOVER_MS}ms ${HOVER_EASE}` },
    '& [data-tile]:hover': { transform: 'scale(1.04)', zIndex: 1 },
    '&:has([data-tile]:hover) [data-tile]:not(:hover)': {
      transform: 'scale(0.96)',
      filter: 'brightness(0.6)',
    },
    '& [data-tile]:hover [data-art]': { transform: 'scale(1.12) translateX(-2%)' },
    '&:has([data-tile]:hover) [data-tile]:not(:hover) [data-art]': {
      transform: 'scale(1.12) translateX(2%)',
    },
  },
} as const

export function GameSelect() {
  const { t } = useLingui()
  const { status, loadError, refresh, tileProps } = useGameTiles()
  if (loadError) {
    return <LoadErrorRow error={loadError} onRetry={refresh} />
  }
  if (!status) {
    return <LoadingRow>{t`Loading…`}</LoadingRow>
  }
  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          display: 'grid',
          gridTemplateColumns: `repeat(auto-fill, minmax(min(100%, ${TILE_MIN_WIDTH_PX}px), 1fr))`,
          gridAutoRows: `minmax(${TILE_MIN_PX}px, 1fr)`,
          ...hoverFocus,
        }}
      >
        {ordered(status.games).map((g) => (
          <Row key={g.id} {...tileProps(g)} />
        ))}
      </Box>
      {!status.games.some((g) => g.available && g.installed) && (
        <Box sx={{ px: 2, py: 0.75, display: 'flex', alignItems: 'center', gap: 1 }}>
          <Typography sx={{ fontSize: 14, color: 'text.secondary' }}>
            {t`No supported game was found in your launchers.`}
          </Typography>
          <Link component="button" onClick={() => openSettings('launchers')} sx={{ fontSize: 14 }}>
            {t`Check launchers…`}
          </Link>
        </Box>
      )}
    </Box>
  )
}
