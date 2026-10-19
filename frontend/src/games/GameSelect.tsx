import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useVirtualizer } from '@tanstack/react-virtual'
import { Eye, EyeOff, Play, Settings } from 'lucide-react'
import { type MouseEvent, memo, useCallback, useRef, useState } from 'react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Played } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import {
  SetGameHidden,
  SetLastGame,
  SetLastProfile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { hasSourceLogo } from '../brand/sources/sourceIcons.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useNow } from '../i18n/useNow.ts'
import { playDirect } from '../launch/directPref.ts'
import { useLaunch } from '../launch/store.ts'
import { type GameId, useNav } from '../nav/store.ts'
import { useSettings } from '../settings/store.ts'
import { arrowFocus } from '../shell/arrowFocus.ts'
import { CoverButton } from '../shell/CoverButton.tsx'
import { LoadErrorRow, LoadingRow } from '../shell/LoadingRow.tsx'
import { Tooltip } from '../shell/Tooltip.tsx'
import { reportError } from '../toasts/report.ts'
import { gameArt } from './art.ts'
import { listItems, useFindGame, useTypeToScroll } from './gameList.ts'
import { listedGames } from './hidden.ts'
import { LetterRail } from './LetterRail.tsx'
import { ListFooter } from './ListFooter.tsx'
import { gameOrder, initialOf } from './order.ts'
import { ProfileCards } from './ProfileCards.tsx'
import { formatPlaytime } from './playtime.ts'
import { hoverFocus, listItemSx, ROW_PX } from './rowStyles.ts'
import { storeName } from './storeName.ts'
import { useGameTiles } from './useGameTiles.ts'
import { useOpenGame } from './useOpenGame.ts'

type Game = GameInfo

const ART_OPACITY = 0.72
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
        // Art below the fold waits until its row scrolls near, and art the store does not have leaves the row's
        // dark base showing in place of a broken-image mark.
        loading="lazy"
        decoding="async"
        // Shown once whole: a half-loaded picture paints its top over a black band.
        onLoad={(e) => {
          e.currentTarget.style.opacity = String(ART_OPACITY)
        }}
        onError={(e) => {
          e.currentTarget.style.display = 'none'
        }}
        sx={{
          position: 'absolute',
          inset: 0,
          width: '100%',
          height: '100%',
          objectFit: 'cover',
          opacity: 0,
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

// Logos where a source has one: the names are well known, and the tooltip carries them for anyone who is not sure. A
// source with no logo shows its name in the same badge.
function SourceBadges({ sources }: { sources: string[] }) {
  return (
    <Box sx={{ display: 'flex', gap: '6px' }}>
      {sources.map((id) => (
        <Tooltip key={id} title={sourceLabel(id)}>
          <Box
            role="img"
            aria-label={sourceLabel(id)}
            // Focusable and open to the pointer, or the tooltip that names the logo could never show.
            tabIndex={0}
            sx={{
              pointerEvents: 'auto',
              minWidth: 36,
              height: 36,
              px: hasSourceLogo(id) ? 0 : 1,
              display: 'grid',
              placeItems: 'center',
              bgcolor: 'var(--mortar-game-dim)',
              borderRadius: '6px',
              typography: 'caption',
              fontWeight: 600,
            }}
          >
            {hasSourceLogo(id) ? <SourceLogo id={id} size={22} /> : sourceLabel(id)}
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

interface RowProps {
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
  hidden: boolean
  cards: { gameId: GameId; profiles: Profile[]; lastPlayed: Played | undefined } | undefined
}

function RowView({
  game,
  openable,
  setupNeeded,
  note,
  loader,
  lastPlayedName,
  lastPlayedAt,
  lastPlayedId,
  playtimeMs,
  hidden,
  cards,
}: RowProps) {
  const { t } = useLingui()
  const loaderLine = useLoaderLine({ game, loader, lastPlayedName, lastPlayedAt, playtimeMs })
  const openGameTile = useOpenGame()
  const open = () => openGameTile(game)
  // A game never played yet still gets Play, for its first profile.
  const first = cards?.profiles[0]
  const playId = lastPlayedId || first?.id || ''
  const playName = lastPlayedName || first?.name || ''
  const playLast = usePlayLast(game, playId)
  const art = gameArt(game, globalThis.innerWidth)
  const hasArt = art !== ''
  const content = (
    <>
      {art ? <Art src={art} openable={openable} /> : null}
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
        <Typography sx={{ fontSize: 15, opacity: 0.9 }}>
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
        <Button
          type="button"
          size="small"
          color="inherit"
          startIcon={hidden ? <Eye size={14} /> : <EyeOff size={14} />}
          onClick={() =>
            SetGameHidden(game.id, !hidden).catch((err: unknown) =>
              fail(t`Could not save that setting`, err),
            )
          }
        >
          {hidden ? t`Unhide` : t`Hide this game`}
        </Button>
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
    <Box data-tile="" data-game={game.id} sx={sx}>
      {content}
    </Box>
  )
}

// The count says how many games the group holds, which the rows below only show by scrolling.
// tileProps builds a fresh cards object each render; a tile renders again only when what it shows changed.
const sameCards = (a: RowProps['cards'], b: RowProps['cards']) =>
  a === b ||
  (a !== undefined &&
    b !== undefined &&
    a.gameId === b.gameId &&
    a.profiles === b.profiles &&
    a.lastPlayed === b.lastPlayed)

const Row = memo(
  RowView,
  (a, b) =>
    a.game === b.game &&
    a.openable === b.openable &&
    a.setupNeeded === b.setupNeeded &&
    a.note === b.note &&
    a.loader === b.loader &&
    a.lastPlayedName === b.lastPlayedName &&
    a.lastPlayedAt === b.lastPlayedAt &&
    a.lastPlayedId === b.lastPlayedId &&
    a.playtimeMs === b.playtimeMs &&
    a.hidden === b.hidden &&
    sameCards(a.cards, b.cards),
)

const GroupHeading = memo(function GroupHeadingView({
  children,
  count,
}: {
  children: string
  count: number
}) {
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
      <Box
        component="span"
        sx={{ ml: 1, color: 'var(--mortar-ink-dim-60)', fontVariantNumeric: 'tabular-nums' }}
      >
        {count}
      </Box>
    </Typography>
  )
})

// A shorter list fits without a rail to jump through it.
const RAIL_MIN_GAMES = 8

const HEADING_PX = 40

// The rail is memoised, so it gets one function that always calls the latest jump.
function useLatestJump() {
  const jumpRef = useRef<(letter: string) => void>(() => undefined)
  const onJump = useCallback((letter: string) => jumpRef.current(letter), [])
  return { jumpRef, onJump }
}

function GameSelect() {
  const { t } = useLingui()
  const { status, loadError, refresh, tileProps } = useGameTiles()
  const lastGame = useSettings((s) => s.lastGame)
  const played = useSettings((s) => s.lastPlayed)
  const growRows = useSettings((s) => s.growGameRows) !== false
  const scroller = useRef<HTMLElement | null>(null)
  const [current, setCurrent] = useState('')
  const hiddenIds = useSettings((s) => s.hiddenGames)
  const [showHidden, setShowHidden] = useState(false)
  const { jumpRef, onJump } = useLatestJump()
  const ordered = gameOrder(
    listedGames(status?.games ?? [], hiddenIds, showHidden),
    lastGame,
    played,
  )
  const { recent } = ordered
  // Games found on this computer come before the rest of the catalog, most of which is not set up here.
  const found = ordered.rest.filter((g) => g.available && g.installed)
  const rest = ordered.rest.filter((g) => !(g.available && g.installed))
  const items = listItems([
    { label: t`Recently played`, games: recent },
    { label: t`On this computer`, games: found },
    { label: recent.length + found.length > 0 ? t`Other games` : t`All games`, games: rest },
  ])
  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => scroller.current,
    estimateSize: (i) => (items[i]?.kind === 'heading' ? HEADING_PX : ROW_PX),
    overscan: 3,
    getItemKey: (i) => items[i]?.key ?? i,
  })
  // Brings a game's row to the top of the list and focuses it once rendered, so Enter opens it.
  const reveal = (id: string) => {
    const index = items.findIndex((it) => it.kind === 'game' && it.game.id === id)
    if (index < 0) {
      return
    }
    virtualizer.scrollToIndex(index, { align: 'start' })
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        scroller.current
          ?.querySelector<HTMLElement>(`[data-tile][data-game="${CSS.escape(id)}"] button`)
          ?.focus({ preventScroll: true })
      })
    })
  }
  const findGame = useFindGame({
    hidden: hiddenIds,
    showHidden,
    setShowHidden,
    listed: (id) => items.some((it) => it.kind === 'game' && it.game.id === id),
    reveal,
  })
  useTypeToScroll(status?.games.filter((g) => g.available) ?? [], findGame)
  if (loadError) {
    return <LoadErrorRow error={loadError} onRetry={refresh} />
  }
  if (!status) {
    return <LoadingRow>{t`Loading…`}</LoadingRow>
  }
  const rail = rest.length >= RAIL_MIN_GAMES
  const letters = [...new Set(rest.map((g) => initialOf(g.name)))]
  // The rail marks the letter of the first alphabetical row still in view.
  const restIds = new Set(rest.map((g) => g.id))
  // At the end of the list the last letters' rows all fit on one screen, so the bottom marks the last letter.
  const markLetter = () => {
    const box = scroller.current
    const offset = box?.scrollTop ?? 0
    const atEnd = box ? offset + box.clientHeight >= box.scrollHeight - 1 : false
    const shown = virtualizer.getVirtualItems().find((v) => {
      const it = items[v.index]
      return v.end > offset + 1 && it?.kind === 'game' && restIds.has(it.game.id)
    })
    const it = shown ? items[shown.index] : undefined
    const inView = it?.kind === 'game' ? it.game : rest[0]
    const first = atEnd ? rest.at(-1) : inView
    setCurrent(first && offset > 0 ? initialOf(first.name) : '')
  }
  const jump = (letter: string) => {
    const first = rest.find((g) => initialOf(g.name) === letter)
    if (first) {
      reveal(first.id)
    }
  }
  jumpRef.current = jump
  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ flex: 1, minHeight: 0, display: 'flex' }}>
        <Box
          ref={scroller}
          sx={{
            flex: 1,
            minWidth: 0,
            overflowY: 'auto',
            overflowX: 'hidden',
            // The rail stands in for the scrollbar.
            ...(rail
              ? { scrollbarWidth: 'none', '&::-webkit-scrollbar': { display: 'none' } }
              : {}),
            ...(growRows ? hoverFocus : {}),
          }}
          onScroll={rail ? markLetter : undefined}
          onKeyDown={(e) => arrowFocus(e, '[data-tile]')}
        >
          <Box sx={{ position: 'relative', width: '100%', height: virtualizer.getTotalSize() }}>
            {virtualizer.getVirtualItems().map((v) => {
              const it = items[v.index]
              return (
                <Box
                  key={v.key}
                  sx={listItemSx}
                  style={{ height: v.size, transform: `translateY(${v.start}px)` }}
                >
                  {it?.kind === 'heading' ? (
                    <GroupHeading count={it.count}>{it.label}</GroupHeading>
                  ) : null}
                  {it?.kind === 'game' ? <Row {...tileProps(it.game)} /> : null}
                </Box>
              )
            })}
          </Box>
        </Box>
        {rail ? <LetterRail letters={letters} current={current} onJump={onJump} /> : null}
      </Box>
      <ListFooter
        games={status.games}
        hiddenCount={hiddenIds?.length ?? 0}
        showHidden={showHidden}
        onShowHidden={setShowHidden}
        onChanged={refresh}
      />
    </Box>
  )
}

export { GameSelect }
