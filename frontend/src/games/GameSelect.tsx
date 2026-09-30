import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { type FocusEvent, type PointerEvent, useEffect, useState } from 'react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import { List } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { SetLastGame } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { useNav } from '../nav/store.ts'
import { useToasts } from '../toasts/store.ts'
import { type GameStatus, loadGameStatus } from './status.ts'

type Game = GameInfo

const SOURCES: Record<string, string[]> = {
  stardew: ['Nexus', 'GitHub'],
  lethal: ['Thunderstore'],
}

const LONG_NAME = 8
const SMALL_FONT = 13
const NORMAL_FONT = 15
const EASE = '700ms cubic-bezier(0.4, 0, 0.2, 1)'
const EXPANDED_SHARE = 62
const shadow = '0 1px 2px rgba(0,0,0,0.9), 0 0 18px rgba(0,0,0,0.85)'

// Parallax input for the art: -1 at the row's top edge to 1 at its bottom, set straight on
// the element so pointer movement never re-renders React.
function trackPointer(e: PointerEvent<HTMLElement>) {
  const r = e.currentTarget.getBoundingClientRect()
  e.currentTarget.style.setProperty('--py', String(((e.clientY - r.top) / r.height) * 2 - 1))
}

function Row({
  game,
  openable,
  note,
  share,
  expanded,
  onActive,
}: {
  game: Game
  openable: boolean
  note: string
  share: number
  expanded: boolean
  onActive: (on: boolean) => void
}) {
  const { t } = useLingui()
  const { loader } = game
  const open = () => {
    if (game.id !== 'stardew') {
      return
    }
    SetLastGame(game.id).catch((e: unknown) =>
      useToasts
        .getState()
        .push({ kind: 'error', title: t`Could not save the last game`, body: String(e) }),
    )
    useNav.getState().openGame(game.id)
  }
  const content = (
    <>
      {game.artUrl ? (
        <Box
          component="img"
          className="art"
          src={game.artUrl}
          alt=""
          sx={{
            position: 'absolute',
            left: 0,
            top: '-15%',
            width: '100%',
            height: '130%',
            objectFit: 'cover',
            opacity: 0.72,
            filter: `${openable ? '' : 'saturate(0.6) '}${expanded ? 'brightness(1.05) contrast(1.1)' : 'brightness(0.8) contrast(1)'}`,
            scale: expanded ? 1.04 : 1,
            transform: 'translateY(calc(var(--py, 0) * -10%))',
            transition: `filter ${EASE}, scale ${EASE}, transform 150ms ease-out`,
          }}
        />
      ) : null}
      {game.artUrl ? (
        <Box
          sx={{
            position: 'absolute',
            inset: 0,
            bgcolor: openable ? 'rgba(0,0,0,0.28)' : 'rgba(0,0,0,0.55)',
          }}
        />
      ) : null}
      <Box
        className="text"
        sx={{
          position: 'relative',
          textShadow: shadow,
          textAlign: 'left',
          color: '#fff',
          transform: expanded ? 'translateX(12px)' : 'none',
          transition: `transform ${EASE}`,
        }}
      >
        <Typography sx={{ fontSize: 34, fontWeight: 600, lineHeight: 1.2 }}>{game.name}</Typography>
        <Typography sx={{ fontSize: 17 }}>{t`${loader} | Steam`}</Typography>
        <Typography sx={{ mt: '6px', fontSize: 16, fontWeight: 600 }}>{note}</Typography>
      </Box>
      <Box
        className="badges"
        sx={{
          position: 'relative',
          display: 'flex',
          gap: '14px',
          transform: expanded ? 'translateX(-12px)' : 'none',
          transition: `transform ${EASE}`,
        }}
      >
        {(SOURCES[game.id] ?? []).map((name) => (
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
      </Box>
    </>
  )
  const hover = {
    onPointerEnter: () => onActive(true),
    onPointerLeave: () => onActive(false),
    onFocus: (e: FocusEvent<HTMLElement>) => onActive(e.currentTarget.matches(':focus-visible')),
    onBlur: () => onActive(false),
  }
  const sx = {
    position: 'relative',
    flex: `0 0 ${share}%`,
    minHeight: 72,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    px: '96px',
    overflow: 'hidden',
    bgcolor: game.artUrl ? 'transparent' : 'background.paper',
    borderLeft: `${expanded ? 8 : 4}px solid`,
    borderColor: openable ? 'primary.main' : 'transparent',
    borderTop: '1px solid rgba(0,0,0,0.8)',
    fontFamily: 'inherit',
    transition: `flex-basis ${EASE}, border-left-width ${EASE}`,
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'none',
      '& .art, & .text, & .badges': { transition: 'none' },
      '& .art': { transform: 'none' },
    },
  } as const
  return openable ? (
    <ButtonBase onClick={open} onPointerMove={trackPointer} {...hover} sx={sx}>
      {content}
    </ButtonBase>
  ) : (
    <Box onPointerMove={trackPointer} {...hover} sx={sx}>
      {content}
    </Box>
  )
}

export function GameSelect() {
  const { t } = useLingui()
  const [status, setStatus] = useState<GameStatus | null>(null)
  const [profileCount, setProfileCount] = useState(0)
  const [active, setActive] = useState<string | null>(null)
  useEffect(() => {
    Promise.all([loadGameStatus(), List('stardew')])
      .then(([s, profiles]) => {
        setStatus(s)
        setProfileCount(profiles?.length ?? 0)
      })
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
    if (g.id !== 'stardew') {
      return t`Installed`
    }
    return plural(profileCount, {
      one: 'Installed · # profile',
      other: 'Installed · # profiles',
    })
  }
  const count = status.games.length
  const shareOf = (id: string) => {
    if (active === null || count < 2) {
      return 100 / count
    }
    return id === active ? EXPANDED_SHARE : (100 - EXPANDED_SHARE) / (count - 1)
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
            share={shareOf(g.id)}
            expanded={active === g.id}
            onActive={(on) => setActive((cur) => (on ? g.id : cur === g.id ? null : cur))}
          />
        ))}
      </Box>
      {status.steam !== 'found' && (
        <Typography noWrap={true} sx={{ px: 2, py: 0.75, fontSize: 14, color: 'text.secondary' }}>
          {t`Mortar supports Steam installed directly on the system.`}{' '}
          {status.steam === 'flatpak-only'
            ? t`Only a Flatpak Steam was found.`
            : t`Steam was not found.`}
        </Typography>
      )}
    </Box>
  )
}
