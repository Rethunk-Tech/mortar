import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, MenuItem, Typography } from '@mui/material'
import { ChevronDown } from 'lucide-react'
import { memo } from 'react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import type { Played } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { compact } from '../game/compact.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useNow } from '../i18n/useNow.ts'
import { routeGame, useNav } from '../nav/store.ts'
import { useSettings } from '../settings/store.ts'
import { MenuHeading, MenuRule, TitleMenu, TitleMenuItem } from '../shell/TitleMenu.tsx'
import { useTitleMenu } from '../shell/titleMenus.ts'
import { space } from '../theme/density.ts'
import { gameArt, HERO_ASPECT } from './art.ts'
import { listedGames } from './hidden.ts'
import { useGameInfo, useGames } from './info.ts'
import { gameOrder } from './order.ts'
import { formatPlaytime } from './playtime.ts'
import { useOpenGame } from './useOpenGame.ts'

const THUMB_PX = 22
const MENU_PX = 300
const STRIP_PX = 56

function Thumb({ game }: { game: GameInfo | undefined }) {
  const art = gameArt(game, THUMB_PX * HERO_ASPECT)
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

// One game in the switcher: a strip of its art under a dark tint, its name, and when it was last played.
function GameStrip({
  game,
  played,
  current,
  onClick,
}: {
  game: GameInfo
  played: Played | undefined
  current: boolean
  onClick: () => void
}) {
  const { t, i18n } = useLingui()
  useNow()
  const art = gameArt(game, MENU_PX)
  const playtime = formatPlaytime(played?.playtimeMs ?? 0, i18n.locale)
  const line = [
    played?.at ? t`played ${formatWhen(played.at)}` : '',
    playtime ? t`${playtime} played` : '',
  ]
    .filter(Boolean)
    .join(' · ')
  return (
    <MenuItem
      role="menuitemradio"
      aria-checked={current}
      onClick={onClick}
      sx={{
        position: 'relative',
        overflow: 'hidden',
        height: STRIP_PX,
        mx: 0.75,
        my: 0.25,
        px: '12px',
        borderRadius: '6px',
        borderLeft: '3px solid',
        borderColor: current ? 'primary.main' : 'transparent',
        color: '#fff',
        bgcolor: 'var(--mortar-raised)',
        backgroundImage: art ? `url("${art}")` : 'none',
        backgroundSize: 'cover',
        backgroundPosition: 'center',
        '&::before': {
          content: '""',
          position: 'absolute',
          inset: 0,
          bgcolor: 'rgba(0, 0, 0, 0.58)',
        },
        '&:hover::before, &.Mui-focusVisible::before': { bgcolor: 'rgba(0, 0, 0, 0.42)' },
      }}
    >
      <Box sx={{ position: 'relative', minWidth: 0 }}>
        <Typography noWrap={true} sx={{ fontSize: 14, fontWeight: 700, lineHeight: 1.25 }}>
          {game.name}
        </Typography>
        {line ? (
          <Typography noWrap={true} sx={{ fontSize: 12, lineHeight: 1.3, opacity: 0.85 }}>
            {line}
          </Typography>
        ) : null}
      </Box>
    </MenuItem>
  )
}

// The title bar's game switcher: the open game's art and name (or "Choose a game") over a menu of the playable games.
// Memoised: the title bar re-renders on every navigation, and this one does not need to.
export const GameMenu = memo(function GameMenuView() {
  const { t } = useLingui()
  // The game being set up has no game route yet but is still the one the title bar names.
  const game = useNav(
    (s) => routeGame(s.route) ?? (s.route.name === 'game-setup' ? s.route.game : ''),
  )
  const info = useGameInfo(game)
  const games = useGames()
  const lastGame = useSettings((s) => s.lastGame)
  const played = useSettings((s) => s.lastPlayed)
  const hidden = useSettings((s) => s.hiddenGames)
  const { anchor, close, trigger } = useTitleMenu('game')
  const { recent, rest } = gameOrder(
    listedGames(
      games.filter((g) => g.available),
      hidden,
      false,
    ),
    lastGame,
    played,
  )
  // The rest of the catalog, mostly games not set up here, is one click away under All games.
  const found = rest.filter((g) => g.installed)
  const item = (g: GameInfo) => (
    <GameStrip
      key={g.id}
      game={g}
      played={played?.[g.id]}
      current={g.id === game}
      onClick={() => {
        close()
        openGame(g)
      }}
    />
  )
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
          height: space.control,
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
      <TitleMenu anchorEl={anchor} onClose={close} label={t`Games`} width={MENU_PX}>
        {recent.length > 0 ? <MenuHeading>{t`Recently played`}</MenuHeading> : null}
        {recent.map(item)}
        {found.length > 0 ? <MenuHeading>{t`On this computer`}</MenuHeading> : null}
        {found.map(item)}
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
})
