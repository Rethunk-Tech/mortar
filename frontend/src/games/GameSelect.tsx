import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonBase, Typography } from '@mui/material'
import { Play } from 'lucide-react'
import { type MouseEvent, useEffect, useState } from 'react'
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
import { useLaunch } from '../launch/store.ts'
import { useLoader } from '../loader/store.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { gameArt } from './art.ts'
import { relativePlay } from './lastPlayed.ts'
import { type GameStatus, loaderCaption, loadGameStatus } from './status.ts'

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
  useToasts.getState().push({ kind: 'error', title, body: errorMessage(err) })
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
  const { t } = useLingui()
  const start = useLaunch((s) => s.start)
  const rel = relativePlay(lastPlayedAt, Date.now())
  let ago = ''
  if (rel?.kind === 'now') {
    ago = t`just now`
  } else if (rel?.unit === 'minute') {
    ago = plural(rel.n, { one: '# minute ago', other: '# minutes ago' })
  } else if (rel?.unit === 'hour') {
    ago = plural(rel.n, { one: '# hour ago', other: '# hours ago' })
  } else if (rel?.unit === 'day') {
    ago = plural(rel.n, { one: '# day ago', other: '# days ago' })
  }
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
    start(game.id, lastPlayedId, false).then(() => undefined)
  }
  const storeName = (id: string) => {
    if (id === 'flatpak-steam') {
      return t`Flatpak Steam`
    }
    if (id === 'gog') {
      return t`GOG`
    }
    if (id === 'gog-heroic') {
      return t`GOG via Heroic`
    }
    if (id === 'lutris') {
      return t`Lutris`
    }
    return t`Steam`
  }
  const store = game.store ? storeName(game.store) : ''
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
        <Typography sx={{ fontSize: 17 }}>{loaderLine}</Typography>
        <Typography sx={{ mt: '6px', fontSize: 16, fontWeight: 600 }}>{note}</Typography>
      </Box>
      <Box sx={{ position: 'relative', display: 'flex', alignItems: 'center', gap: '14px' }}>
        {openable && lastPlayedId ? (
          <Button
            component="span"
            variant="contained"
            size="large"
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
    <ButtonBase onClick={open} sx={sx}>
      {content}
    </ButtonBase>
  ) : (
    <Box sx={sx}>{content}</Box>
  )
}

export function GameSelect() {
  const { t } = useLingui()
  const [status, setStatus] = useState<GameStatus | null>(null)
  const [profiles, setProfiles] = useState<Profile[]>([])
  const [lastPlayedId, setLastPlayedId] = useState('')
  const [lastPlayedAt, setLastPlayedAt] = useState('')
  const loaderStatus = useLoader((s) => s.status)
  const checkLoader = useLoader((s) => s.check)
  useEffect(() => {
    checkLoader('stardew')
  }, [checkLoader])
  useEffect(() => {
    Promise.all([loadGameStatus(), List('stardew'), Get()])
      .then(([s, listed, settings]) => {
        setStatus(s)
        const next = listed ?? []
        setProfiles(next)
        const played = settings.lastPlayed?.stardew
        const still = Boolean(played?.profile && next.some((p) => p.id === played.profile))
        setLastPlayedId(still && played ? played.profile : '')
        setLastPlayedAt(still && played ? played.at : '')
      })
      .catch((err: unknown) => fail(t`Could not read your games`, err))
  }, [t])
  if (!status) {
    return null
  }
  const lastName = profiles.find((p) => p.id === lastPlayedId)?.name ?? ''
  const noteFor = (g: Game) => {
    if (!g.available) {
      return t`After the first release`
    }
    if (!g.installed) {
      return t`Not found`
    }
    if (g.id !== 'stardew') {
      return t`Installed`
    }
    return plural(profiles.length, {
      one: 'Installed · # profile',
      other: 'Installed · # profiles',
    })
  }
  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        {status.games.map((g) => (
          <Row
            key={g.id}
            game={g}
            openable={g.available && g.installed}
            note={noteFor(g)}
            loader={loaderCaption(g.loader, g.id === 'stardew' ? loaderStatus : null)}
            lastPlayedName={g.id === 'stardew' ? lastName : ''}
            lastPlayedAt={g.id === 'stardew' ? lastPlayedAt : ''}
            lastPlayedId={g.id === 'stardew' ? lastPlayedId : ''}
          />
        ))}
      </Box>
      {status.steam !== 'found' && !status.games.some((g) => g.available && g.installed) && (
        <Typography noWrap={true} sx={{ px: 2, py: 0.75, fontSize: 14, color: 'text.secondary' }}>
          {t`Mortar looks for Stardew Valley in Steam, Flatpak Steam, GOG, Heroic and Lutris.`}{' '}
          {status.steam === 'flatpak-only'
            ? t`Only a Flatpak Steam was found, with no game in its library.`
            : t`Steam was not found.`}
        </Typography>
      )}
    </Box>
  )
}
