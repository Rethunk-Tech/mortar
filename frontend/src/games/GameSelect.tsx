import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link, Tooltip, Typography } from '@mui/material'
import { Play, Settings } from 'lucide-react'
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
import { formatWhen } from '../i18n/formatWhen.ts'
import { useNow } from '../i18n/useNow.ts'
import { absoluteWhen } from '../i18n/when.ts'
import { playDirect } from '../launch/directPref.ts'
import { useLaunch } from '../launch/store.ts'
import { type GameId, openSettings, useNav } from '../nav/store.ts'
import { arrowFocus } from '../shell/arrowFocus.ts'
import { CoverButton } from '../shell/CoverButton.tsx'
import { LoadErrorRow, LoadingRow } from '../shell/LoadingRow.tsx'
import { reportError } from '../toasts/report.ts'
import { gameArt } from './art.ts'
import { ProfileCards } from './ProfileCards.tsx'
import { formatPlaytime } from './playtime.ts'
import { storeName } from './storeName.ts'
import { ordered, useGameTiles } from './useGameTiles.ts'
import { useOpenGame } from './useOpenGame.ts'

type Game = GameInfo

// A short catalog reads as a list of full-width rows; past this many games the rows wrap into columns.
const LIST_MAX_GAMES = 4
const ROW_MIN_PX = 168
const ROW_MAX_PX = 260
const TILE_MIN_WIDTH_PX = 560
const HOVER_MS = 200
const HOVER_EASE = 'cubic-bezier(0.2, 0.8, 0.2, 1)'
const shadow = '0 1px 2px var(--mortar-overlay-90), 0 0 18px var(--mortar-overlay-85)'
const actionSx = {
  minWidth: 132,
  height: 44,
  fontSize: 16,
  fontWeight: 700,
} as const

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

// Logos only: the names are well known, and the tooltip carries them for anyone who is not sure.
function SourceBadges({ sources }: { sources: string[] }) {
  return (
    <Box sx={{ display: 'flex', gap: '6px' }}>
      {sources.map((id) => (
        <Tooltip key={id} title={sourceLabel(id)}>
          <Box
            role="img"
            aria-label={sourceLabel(id)}
            sx={{
              width: 36,
              height: 36,
              display: 'grid',
              placeItems: 'center',
              bgcolor: 'var(--mortar-game-dim)',
              borderRadius: '6px',
            }}
          >
            <SourceLogo id={id} size={22} />
          </Box>
        </Tooltip>
      ))}
    </Box>
  )
}

function useLoaderLine({
  game,
  loader,
  lastPlayedName,
  lastPlayedAt,
  playtimeMs,
}: {
  game: Game
  loader: string
  lastPlayedName: string
  lastPlayedAt: string
  playtimeMs: number
}) {
  const { t, i18n } = useLingui()
  useNow()
  const ago = lastPlayedName ? formatWhen(lastPlayedAt) : ''
  const playtime = formatPlaytime(playtimeMs, i18n.locale)
  const named = game.store ? storeName(game.store) : null
  return [loader, named ? t(named) : '', ago, playtime ? t`${playtime} played` : '']
    .filter(Boolean)
    .join(' · ')
}

function usePlayLast(game: Game, lastPlayedId: string) {
  const { t } = useLingui()
  const start = useLaunch((s) => s.start)
  return (ev: MouseEvent) => {
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
}

function Row({
  game,
  openable,
  setupNeeded,
  note,
  loader,
  lastPlayedName,
  lastPlayedAt,
  lastPlayedId,
  playtimeMs,
  cards,
}: {
  game: Game
  openable: boolean
  // Available but never opened: the tile offers Set up where a set-up game offers Play.
  setupNeeded: boolean
  note: string
  loader: string
  lastPlayedName: string
  lastPlayedAt: string
  lastPlayedId: string
  playtimeMs: number
  cards: { gameId: GameId; profiles: Profile[]; lastPlayed: Played | undefined } | undefined
}) {
  const { t, i18n } = useLingui()
  const loaderLine = useLoaderLine({ game, loader, lastPlayedName, lastPlayedAt, playtimeMs })
  const openGameTile = useOpenGame()
  const open = () => openGameTile(game)
  // A game never played yet still gets Play, for its first profile.
  const first = cards?.profiles[0]
  const playId = lastPlayedId || first?.id || ''
  const playName = lastPlayedName || first?.name || ''
  const playLast = usePlayLast(game, playId)
  const hasArt = gameArt(game) !== ''
  const content = (
    <>
      {gameArt(game) ? <Art src={gameArt(game)} openable={openable} /> : null}
      {openable ? <CoverButton onClick={open} aria-label={t`Open ${game.name}`} /> : null}
      <Box
        sx={{
          ...aboveOpen,
          flex: 1,
          minWidth: 0,
          overflow: 'hidden',
          textShadow: hasArt ? shadow : 'none',
          textAlign: 'left',
          color: hasArt ? 'common.white' : 'text.primary',
          display: 'flex',
          flexDirection: 'column',
          gap: '4px',
        }}
      >
        <Typography sx={{ fontSize: 30, fontWeight: 600, lineHeight: 1.15 }}>
          {game.name}
        </Typography>
        <Typography
          title={lastPlayedAt ? absoluteWhen(lastPlayedAt, i18n.locale) || undefined : undefined}
          sx={{ fontSize: 15, opacity: 0.9 }}
        >
          {[loaderLine, note].filter(Boolean).join(' · ')}
        </Typography>
        {cards ? (
          <ProfileCards
            gameId={cards.gameId}
            gameName={game.name}
            profiles={cards.profiles}
            lastPlayed={cards.lastPlayed}
          />
        ) : null}
      </Box>
      <Box
        sx={{
          ...aboveOpen,
          flexShrink: 0,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-end',
          gap: '12px',
        }}
      >
        <SourceBadges sources={game.sources ?? []} />
        {openable && playId ? (
          <Button
            type="button"
            variant="contained"
            size="large"
            aria-label={t`Play ${playName}`}
            startIcon={<Play size={18} fill="currentColor" />}
            onClick={playLast}
            sx={actionSx}
          >
            {t`Play`}
          </Button>
        ) : null}
        {setupNeeded ? (
          <Button
            type="button"
            variant="contained"
            size="large"
            aria-label={t`Set up ${game.name}`}
            startIcon={<Settings size={18} />}
            onClick={open}
            sx={actionSx}
          >
            {t`Set up`}
          </Button>
        ) : null}
      </Box>
    </>
  )
  const sx = {
    position: 'relative',
    // Name and profiles on the left, sources and Play on the right, so every row lines its actions up the same way.
    display: 'flex',
    alignItems: 'center',
    gap: '24px',
    px: '40px',
    py: '20px',
    overflow: 'hidden',
    // Art tiles keep a solid dark base so white text stays readable while or after the image fails to load.
    bgcolor: hasArt ? 'var(--mortar-overlay-90)' : 'background.paper',
    borderLeft: '4px solid',
    borderColor: openable ? 'primary.main' : 'transparent',
    borderRadius: '8px',
    fontFamily: 'inherit',
  } as const
  return (
    <Box data-tile="" sx={sx}>
      {content}
    </Box>
  )
}

// Hovering a row drifts its art; the rows themselves never move, so the list stays aligned under the pointer.
const hoverFocus = {
  '@media (hover: hover) and (prefers-reduced-motion: no-preference)': {
    '& [data-art]': { transition: `transform ${HOVER_MS}ms ${HOVER_EASE}` },
    '& [data-tile]:hover [data-art]': { transform: 'scale(1.1) translateX(-2%)' },
  },
} as const

function GameSelect() {
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
          overflowX: 'hidden',
          // Room for the hovered tile's growth, so it is not cut at the grid's edge.
          p: '8px 16px',
          display: 'grid',
          gridTemplateColumns:
            status.games.length <= LIST_MAX_GAMES
              ? '1fr'
              : `repeat(auto-fit, minmax(min(100%, ${TILE_MIN_WIDTH_PX}px), 1fr))`,
          gridAutoRows: `minmax(${ROW_MIN_PX}px, ${ROW_MAX_PX}px)`,
          alignContent: 'start',
          gap: '10px',
          ...hoverFocus,
        }}
        onKeyDown={(e) => arrowFocus(e, '[data-tile]')}
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

export { GameSelect }
