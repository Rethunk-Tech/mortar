import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import { ChevronDown } from 'lucide-react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { compact } from '../game/compact.ts'
import { routeGame, useNav } from '../nav/store.ts'
import { useSettings } from '../settings/store.ts'
import { MenuHeading, MenuRule, TitleMenu, TitleMenuItem } from '../shell/TitleMenu.tsx'
import { useTitleMenu } from '../shell/titleMenus.ts'
import { gameArt } from './art.ts'
import { useGameInfo, useGames } from './info.ts'
import { useOpenGame } from './useOpenGame.ts'

const THUMB_PX = 22
const NEWEST = '￿'

function Thumb({ game }: { game: GameInfo | undefined }) {
  const art = gameArt(game)
  return (
    <Box
      component={art ? 'img' : 'span'}
      {...(art ? { src: art, alt: '', draggable: false } : {})}
      aria-hidden={true}
      sx={{
        width: THUMB_PX,
        height: THUMB_PX,
        flexShrink: 0,
        borderRadius: '5px',
        display: 'block',
        objectFit: 'cover',
        bgcolor: 'var(--mortar-raised)',
      }}
    />
  )
}

// The title bar's game switcher: the open game's art and name (or "Choose a game") over a menu of the playable games.
export function GameMenu() {
  const { t } = useLingui()
  // The game being set up has no game route yet but is still the one the title bar names.
  const game = useNav(
    (s) => routeGame(s.route) ?? (s.route.name === 'game-setup' ? s.route.game : ''),
  )
  const info = useGameInfo(game)
  const games = useGames()
  const lastGame = useSettings((s) => s.lastGame)
  const played = useSettings((s) => s.lastPlayed)
  const { anchor, close, trigger } = useTitleMenu('game')
  const at = (id: string) => (id === lastGame ? NEWEST : (played?.[id]?.at ?? ''))
  const playable = games.filter((g) => g.available).sort((a, b) => at(b.id).localeCompare(at(a.id)))
  const gameLabel = info?.name ?? ''
  const openGame = useOpenGame()
  return (
    <>
      <ButtonBase
        aria-label={game ? t`Switch game: ${gameLabel}` : t`Choose a game`}
        data-tour="game-tab"
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        {...trigger}
        sx={{
          '--wails-draggable': 'no-drag',
          gap: '8px',
          height: 34,
          minWidth: 0,
          px: '10px',
          borderRadius: '8px',
          fontFamily: 'inherit',
          fontSize: 14,
          fontWeight: 600,
          color: 'inherit',
          '&:hover, &[aria-expanded="true"]': { bgcolor: 'var(--mortar-hairline-faint)' },
        }}
      >
        {game ? <Thumb game={info} /> : null}
        <Box
          component="span"
          sx={{
            minWidth: 0,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            [compact]: { maxWidth: 120 },
          }}
        >
          {game ? gameLabel : t`Choose a game`}
        </Box>
        <ChevronDown size={14} aria-hidden={true} style={{ flexShrink: 0 }} />
      </ButtonBase>
      <TitleMenu anchorEl={anchor} onClose={close} label={t`Games`} width={300}>
        <MenuHeading>{t`Games`}</MenuHeading>
        {playable.map((g) => (
          <TitleMenuItem
            key={g.id}
            icon={<Thumb game={g} />}
            label={g.name}
            checked={g.id === game}
            onClick={() => {
              close()
              openGame(g)
            }}
          />
        ))}
        <MenuRule />
        <TitleMenuItem
          label={t`All games…`}
          onClick={() => {
            close()
            useNav.getState().openGameSelect()
          }}
        />
      </TitleMenu>
    </>
  )
}
