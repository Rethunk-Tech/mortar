import { Trans, useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import { SetLastGame } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useNav } from '../nav/store.ts'
import { useToasts } from '../toasts/store.ts'
import { type GameStatus, loadGameStatus } from './status.ts'

type Game = GameStatus['games'][number]

const SOURCES: Record<string, string[]> = {
  stardew: ['Nexus', 'GitHub'],
  lethal: ['Thunderstore'],
}

const shadow = '0 1px 2px rgba(0,0,0,0.9), 0 0 18px rgba(0,0,0,0.85)'

function Row({ game, openable, note }: { game: Game; openable: boolean; note: string }) {
  const open = () => {
    if (game.id !== 'stardew') {
      return
    }
    SetLastGame(game.id).catch((e: unknown) =>
      useToasts
        .getState()
        .push({ kind: 'error', title: 'Could not save the last game', body: String(e) }),
    )
    useNav.getState().openGame(game.id)
  }
  const content = (
    <>
      {game.artUrl ? (
        <Box
          component="img"
          src={game.artUrl}
          alt=""
          sx={{
            position: 'absolute',
            inset: 0,
            width: '100%',
            height: '100%',
            objectFit: 'cover',
            filter: openable ? 'none' : 'saturate(0.6)',
          }}
        />
      ) : null}
      <Box
        sx={{
          position: 'absolute',
          inset: 0,
          bgcolor: openable ? 'rgba(0,0,0,0.28)' : 'rgba(0,0,0,0.55)',
        }}
      />
      <Box sx={{ position: 'relative', textShadow: shadow, textAlign: 'left', color: '#fff' }}>
        <Typography sx={{ fontSize: 34, fontWeight: 600, lineHeight: 1.2 }}>{game.name}</Typography>
        <Typography sx={{ fontSize: 17 }}>
          {game.loader} <span style={{ opacity: 0.6 }}>|</span> Steam
        </Typography>
        <Typography sx={{ mt: '6px', fontSize: 16, fontWeight: 600 }}>{note}</Typography>
      </Box>
      <Box sx={{ position: 'relative', display: 'flex', gap: '14px' }}>
        {(SOURCES[game.id] ?? []).map((name) => (
          <Box
            key={name}
            sx={{
              width: 96,
              height: 96,
              display: 'grid',
              placeItems: 'center',
              bgcolor: 'rgba(28,28,32,0.92)',
              fontSize: name.length > 8 ? 13 : 15,
              fontWeight: 700,
              color: '#fff',
            }}
          >
            {name}
          </Box>
        ))}
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
    bgcolor: 'rgb(28,28,32)',
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
  useEffect(() => {
    loadGameStatus()
      .then(setStatus)
      .catch((e: unknown) =>
        useToasts
          .getState()
          .push({ kind: 'error', title: t`Could not read your games`, body: String(e) }),
      )
  }, [t])
  if (!status) {
    return null
  }
  const noteFor = (g: Game) => {
    if (!g.available) {
      return t`After the first release`
    }
    if (!g.installed) {
      return t`Not found in your Steam library`
    }
    return t`Installed`
  }
  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        {status.games.map((g) => (
          <Row key={g.id} game={g} openable={g.available && g.installed} note={noteFor(g)} />
        ))}
      </Box>
      {status.steam !== 'found' && (
        <Typography noWrap={true} sx={{ px: 2, py: 0.75, fontSize: 14, color: 'text.secondary' }}>
          <Trans>Mortar supports Steam installed directly on the system.</Trans>{' '}
          {status.steam === 'flatpak-only' ? (
            <Trans>Only a Flatpak Steam was found.</Trans>
          ) : (
            <Trans>Steam was not found.</Trans>
          )}
        </Typography>
      )}
    </Box>
  )
}
