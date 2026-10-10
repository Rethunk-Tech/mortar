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
import { useSettings } from '../settings/store.ts'
import { arrowFocus } from '../shell/arrowFocus.ts'
import { CoverButton } from '../shell/CoverButton.tsx'
import { LoadErrorRow, LoadingRow } from '../shell/LoadingRow.tsx'
import { reportError } from '../toasts/report.ts'
import { gameArt } from './art.ts'
import { gameOrder } from './order.ts'
import { ProfileCards } from './ProfileCards.tsx'
import { formatPlaytime } from './playtime.ts'
import { storeName } from './storeName.ts'
import { useGameTiles } from './useGameTiles.ts'
import { useOpenGame } from './useOpenGame.ts'

type Game = GameInfo

const ROW_PX = 168
// How much taller the row under the pointer, or holding focus, stands.
const ROW_GROW_PX = 72
const HOVER_MS = 450
const HOVER_EASE = 'ease-in-out'
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
    minHeight: ROW_PX,
    flexShrink: 0,
    overflow: 'hidden',
    // Art tiles keep a solid dark base so white text stays readable while or after the image fails to load.
    bgcolor: hasArt ? 'var(--mortar-overlay-90)' : 'background.paper',
    fontFamily: 'inherit',
  } as const
  return (
    <Box data-tile="" sx={sx}>
      {content}
    </Box>
  )
}

// The row under the pointer, or holding focus, grows taller and shows more of its art.
const hoverFocus = {
  // Keyboard focus only: a clicked Play button keeps focus, which would leave its row grown under a pointer elsewhere.
  '& [data-tile]:has(:focus-visible)': { minHeight: ROW_PX + ROW_GROW_PX },
  '@media (hover: hover)': {
    '& [data-tile]:hover': { minHeight: ROW_PX + ROW_GROW_PX },
  },
  '@media (prefers-reduced-motion: no-preference)': {
    '& [data-tile]': { transition: `min-height ${HOVER_MS}ms ${HOVER_EASE}` },
  },
} as const

function GroupHeading({ children }: { children: string }) {
  return (
    <Typography
      component="h2"
      sx={{
        m: 0,
        px: '40px',
        pt: '14px',
        pb: '8px',
        fontSize: 12,
        fontWeight: 700,
        letterSpacing: '0.08em',
        textTransform: 'uppercase',
        color: 'var(--mortar-ink-sec)',
      }}
    >
      {children}
    </Typography>
  )
}

function GameSelect() {
  const { t } = useLingui()
  const { status, loadError, refresh, tileProps } = useGameTiles()
  const lastGame = useSettings((s) => s.lastGame)
  const played = useSettings((s) => s.lastPlayed)
  if (loadError) {
    return <LoadErrorRow error={loadError} onRetry={refresh} />
  }
  if (!status) {
    return <LoadingRow>{t`Loading…`}</LoadingRow>
  }
  const { recent, rest } = gameOrder(status.games, lastGame, played)
  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          overflowX: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          ...hoverFocus,
        }}
        onKeyDown={(e) => arrowFocus(e, '[data-tile]')}
      >
        {recent.length > 0 ? <GroupHeading>{t`Recently played`}</GroupHeading> : null}
        {recent.map((g) => (
          <Row key={g.id} {...tileProps(g)} />
        ))}
        {recent.length > 0 && rest.length > 0 ? <GroupHeading>{t`All games`}</GroupHeading> : null}
        {rest.map((g) => (
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
