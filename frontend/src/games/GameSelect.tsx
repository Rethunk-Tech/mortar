import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link, Typography } from '@mui/material'
import { Play } from 'lucide-react'
import { type MouseEvent, useCallback, useEffect, useState } from 'react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { List } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import type { Played } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import {
  Get,
  SetLastGame,
  SetLastProfile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { gameSetupNeeded } from '../firstrun/needed.ts'
import { useRefreshOnFocus } from '../firstrun/useRefreshOnFocus.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useNow } from '../i18n/useNow.ts'
import { absoluteWhen } from '../i18n/when.ts'
import { playDirect } from '../launch/directPref.ts'
import { useLaunch } from '../launch/store.ts'
import { useLoader } from '../loader/store.ts'
import { type GameId, isGameId, openSettings, useNav } from '../nav/store.ts'
import { useSettings } from '../settings/store.ts'
import { CoverButton } from '../shell/CoverButton.tsx'
import { LoadErrorRow, LoadingRow } from '../shell/LoadingRow.tsx'
import { type InlineError, inlineError, reportError } from '../toasts/report.ts'
import { gameArt } from './art.ts'
import { ProfileCards } from './ProfileCards.tsx'
import { type GameStatus, loaderCaption, loadGameStatus } from './status.ts'
import { storeName } from './storeName.ts'

type Game = GameInfo

const LONG_NAME = 8
// Tiles share the window in a grid that grows with the catalog: two games sit side by side, more wrap into rows.
const TILE_MIN_PX = 240
const TILE_MIN_WIDTH_PX = 560
const SMALL_FONT = 13
const NORMAL_FONT = 15
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

function Row({
  game,
  openable,
  note,
  loader,
  lastPlayedName,
  lastPlayedAt,
  lastPlayedId,
  cards,
}: {
  game: Game
  openable: boolean
  note: string
  loader: string
  lastPlayedName: string
  lastPlayedAt: string
  lastPlayedId: string
  cards: { gameId: GameId; profiles: Profile[]; lastPlayed: Played | undefined } | undefined
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
      {openable ? <CoverButton onClick={open} aria-label={t`Open ${game.name}`} /> : null}
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
        <Typography sx={{ mt: '6px', fontSize: 16, fontWeight: 600 }}>{note}</Typography>
        {cards ? (
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
    minHeight: TILE_MIN_PX,
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
  } as const
  return <Box sx={sx}>{content}</Box>
}

// Installed games first, then supported ones not found, then those coming later; catalog order within each.
function ordered(games: Game[]): Game[] {
  const rank = (g: Game) => {
    if (g.available && g.installed) {
      return 0
    }
    return g.available ? 1 : 2
  }
  return games
    .map((g, i) => ({ g, i }))
    .sort((a, b) => rank(a.g) - rank(b.g) || a.i - b.i)
    .map(({ g }) => g)
}

interface GameState {
  profiles: Profile[]
  lastPlayedId: string
  lastPlayedAt: string
  lastPlayed: Played | undefined
  setupNeeded: boolean
}

export function GameSelect() {
  const { t } = useLingui()
  const [status, setStatus] = useState<GameStatus | null>(null)
  const [states, setStates] = useState<Record<string, GameState>>({})
  const [loadError, setLoadError] = useState<InlineError | null>(null)
  const loaderStatus = useLoader((s) => s.status)
  const checkLoader = useLoader((s) => s.check)
  const lastGame = useSettings((s) => s.lastGame)
  const availableGames = status?.games.filter((g) => g.available) ?? []
  const loaderGame = (availableGames.find((g) => g.id === lastGame) ?? availableGames[0])?.id
  useEffect(() => {
    if (loaderGame) {
      checkLoader(loaderGame)
    }
  }, [checkLoader, loaderGame])
  const refresh = useCallback(() => {
    setLoadError(null)
    Promise.all([loadGameStatus(), Get()])
      .then(async ([s, settings]) => {
        const entries = await Promise.all(
          s.games
            .filter((g) => g.available)
            .map(async (g): Promise<[string, GameState]> => {
              const [listed, setupNeeded] = await Promise.all([List(g.id), gameSetupNeeded(g)])
              const profiles = listed ?? []
              const played = settings.lastPlayed?.[g.id]
              const still = Boolean(
                played?.profile && profiles.some((p) => p.id === played.profile),
              )
              return [
                g.id,
                {
                  profiles,
                  setupNeeded,
                  lastPlayedId: still && played ? played.profile : '',
                  lastPlayedAt: still && played ? played.at : '',
                  lastPlayed: still && played ? played : undefined,
                },
              ]
            }),
        )
        setStatus(s)
        setStates(Object.fromEntries(entries))
      })
      .catch((err: unknown) => {
        fail(t`Could not read your games`, err)
        setLoadError(inlineError(err))
      })
  }, [t])
  useEffect(refresh, [refresh])
  useRefreshOnFocus(refresh)
  if (loadError) {
    return <LoadErrorRow error={loadError} onRetry={refresh} />
  }
  if (!status) {
    return <LoadingRow>{t`Loading…`}</LoadingRow>
  }
  const noteFor = (g: Game) => {
    const st = states[g.id]
    if (!g.available) {
      return t`Coming in a later Mortar version`
    }
    if (!st || st.setupNeeded) {
      return t`Not set up · Open it to set it up`
    }
    return plural(st.profiles.length, {
      one: 'Installed · # profile',
      other: 'Installed · # profiles',
    })
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
        }}
      >
        {ordered(status.games).map((g) => {
          const st = states[g.id]
          const lastId = st?.lastPlayedId ?? ''
          const showCards =
            g.available &&
            g.installed &&
            st &&
            !st.setupNeeded &&
            st.profiles.some((p) => !p.hidden)
          return (
            <Row
              key={g.id}
              game={g}
              openable={g.available}
              note={noteFor(g)}
              loader={loaderCaption(g.loader, g.id === loaderGame ? loaderStatus : null)}
              lastPlayedName={st?.profiles.find((p) => p.id === lastId)?.name ?? ''}
              lastPlayedAt={st?.lastPlayedAt ?? ''}
              lastPlayedId={st?.setupNeeded ? '' : lastId}
              cards={
                showCards && isGameId(g.id)
                  ? { gameId: g.id, profiles: st.profiles, lastPlayed: st.lastPlayed }
                  : undefined
              }
            />
          )
        })}
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
