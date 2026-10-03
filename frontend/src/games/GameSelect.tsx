import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonBase, Link, Typography } from '@mui/material'
import { Play } from 'lucide-react'
import { type MouseEvent, useCallback, useEffect, useState } from 'react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { List } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import {
  Get,
  SetLastGame,
  SetLastProfile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { gameSetupNeeded } from '../firstrun/needed.ts'
import { useRefreshOnFocus } from '../firstrun/useRefreshOnFocus.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { playDirect } from '../launch/directPref.ts'
import { useLaunch } from '../launch/store.ts'
import { useLoader } from '../loader/store.ts'
import { isGameId, openSettings, useNav } from '../nav/store.ts'
import { errorDetails, errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { gameArt } from './art.ts'
import { type GameStatus, loaderCaption, loadGameStatus } from './status.ts'
import { storeName } from './storeName.ts'

type Game = GameInfo

const SOURCES: Record<string, string[]> = {
  stardew: ['Nexus', 'GitHub'],
  lethal: ['Thunderstore'],
}

const LONG_NAME = 8
const SMALL_FONT = 13
const NORMAL_FONT = 15
const shadow = '0 1px 2px rgba(0,0,0,0.9), 0 0 18px rgba(0,0,0,0.85)'

function fail(title: string, err: unknown) {
  useToasts
    .getState()
    .push({ kind: 'error', title, body: errorMessage(err), detail: errorDetails(err) })
}

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
          bgcolor: openable ? 'rgba(0,0,0,0.28)' : 'rgba(0,0,0,0.55)',
        }}
      />
    </>
  )
}

function SourceBadges({ gameId }: { gameId: string }) {
  return (
    <>
      {(SOURCES[gameId] ?? []).map((name) => (
        <Box
          key={name}
          sx={{
            width: 96,
            height: 96,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            bgcolor: 'rgba(28,28,32,0.92)',
            fontSize: name.length > LONG_NAME ? SMALL_FONT : NORMAL_FONT,
            fontWeight: 700,
            color: '#fff',
          }}
        >
          <SourceLogo name={name} size={40} />
          {name}
        </Box>
      ))}
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
}: {
  game: Game
  openable: boolean
  note: string
  loader: string
  lastPlayedName: string
  lastPlayedAt: string
  lastPlayedId: string
}) {
  const { t, i18n } = useLingui()
  const start = useLaunch((s) => s.start)
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
    if (game.id !== 'stardew' || !lastPlayedId) {
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
      <Box sx={{ position: 'relative', textShadow: shadow, textAlign: 'left', color: '#fff' }}>
        <Typography sx={{ fontSize: 34, fontWeight: 600, lineHeight: 1.2 }}>{game.name}</Typography>
        <Typography
          title={
            lastPlayedAt
              ? new Intl.DateTimeFormat(i18n.locale, {
                  dateStyle: 'medium',
                  timeStyle: 'short',
                }).format(new Date(lastPlayedAt))
              : undefined
          }
          sx={{ fontSize: 17 }}
        >
          {loaderLine}
        </Typography>
        <Typography sx={{ mt: '6px', fontSize: 16, fontWeight: 600 }}>{note}</Typography>
      </Box>
      <Box sx={{ position: 'relative', display: 'flex', alignItems: 'center', gap: '14px' }}>
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
              textTransform: 'none',
              boxShadow: 'none',
              whiteSpace: 'nowrap',
              '& .MuiButton-startIcon': { mr: '10px' },
            }}
          >
            {t`Play`}
          </Button>
        ) : null}
        <SourceBadges gameId={game.id} />
      </Box>
    </>
  )
  const sx = {
    position: 'relative',
    flex: '1 1 0',
    minHeight: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    px: '96px',
    overflow: 'hidden',
    bgcolor: gameArt(game) ? 'transparent' : 'background.paper',
    borderLeft: '4px solid',
    borderColor: openable ? 'primary.main' : 'transparent',
    borderTop: '1px solid rgba(0,0,0,0.8)',
    fontFamily: 'inherit',
  } as const
  return openable ? (
    <ButtonBase component="div" onClick={open} aria-label={t`Open ${game.name}`} sx={sx}>
      {content}
    </ButtonBase>
  ) : (
    <Box sx={sx}>{content}</Box>
  )
}

interface GameState {
  profiles: Profile[]
  lastPlayedId: string
  lastPlayedAt: string
  setupNeeded: boolean
}

export function GameSelect() {
  const { t } = useLingui()
  const [status, setStatus] = useState<GameStatus | null>(null)
  const [states, setStates] = useState<Record<string, GameState>>({})
  const loaderStatus = useLoader((s) => s.status)
  const checkLoader = useLoader((s) => s.check)
  useEffect(() => {
    checkLoader('stardew')
  }, [checkLoader])
  const refresh = useCallback(() => {
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
                },
              ]
            }),
        )
        setStatus(s)
        setStates(Object.fromEntries(entries))
      })
      .catch((err: unknown) => fail(t`Could not read your games`, err))
  }, [t])
  useEffect(refresh, [refresh])
  useRefreshOnFocus(refresh)
  if (!status) {
    return null
  }
  const noteFor = (g: Game) => {
    const st = states[g.id]
    if (!g.available) {
      return t`After the first release`
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
      <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        {status.games.map((g) => {
          const st = states[g.id]
          const lastId = st?.lastPlayedId ?? ''
          return (
            <Row
              key={g.id}
              game={g}
              openable={g.available}
              note={noteFor(g)}
              loader={loaderCaption(g.loader, g.id === 'stardew' ? loaderStatus : null)}
              lastPlayedName={st?.profiles.find((p) => p.id === lastId)?.name ?? ''}
              lastPlayedAt={st?.lastPlayedAt ?? ''}
              lastPlayedId={st?.setupNeeded ? '' : lastId}
            />
          )
        })}
      </Box>
      {!status.games.some((g) => g.available && g.installed) && (
        <Box sx={{ px: 2, py: 0.75, display: 'flex', alignItems: 'center', gap: 1 }}>
          <Typography noWrap={true} sx={{ fontSize: 14, color: 'text.secondary' }}>
            {t`No supported game was found in your launchers.`}
          </Typography>
          <Link component="button" onClick={() => openSettings('launchers')} sx={{ fontSize: 14 }}>
            {t`Launchers…`}
          </Link>
        </Box>
      )}
    </Box>
  )
}
