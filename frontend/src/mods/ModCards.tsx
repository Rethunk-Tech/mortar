import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Card, Chip, Typography, useMediaQuery } from '@mui/material'
import { memo, useCallback, useLayoutEffect, useMemo, useRef, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { compact, compactQuery } from '../game/compact.ts'
import { boundShortcut } from '../settings/shortcuts.ts'
import { useSettings } from '../settings/store.ts'
import { space } from '../theme/density.ts'
import { PAD_FOCUS } from '../theme/theme.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { AuthorLink } from './AuthorLink.tsx'
import { actingMods, toggleActing } from './actingMods.ts'
import { formatAuthors } from './authorNormalize.ts'
import { CompatChip } from './CompatChip.tsx'
import { showModId, useDetail } from './detail.ts'
import { ExtraFilesChip } from './ExtraFilesChip.tsx'
import { firstTag } from './group.ts'
import type { ListRow } from './listColumns.ts'
import { entryOf, modId, nexusIdOf } from './lookup.ts'
import { ModMenu } from './ModMenu.tsx'
import { GroupHeaderRow } from './ModsGroupHeader.tsx'
import { useMarked, useTabStop } from './marked.ts'
import { contextMenuProps } from './menu.ts'
import { useNexusFresh } from './nexusDetails.ts'
import { OverlayCountChip } from './OverlayRow.tsx'
import { ownSlice } from './overlayRows.ts'
import {
  LastRunBadge,
  LetterTile,
  LiveBadge,
  NexusGoneBadge,
  PinBadge,
  ProblemBadge,
  UpdateBadge,
} from './parts.tsx'
import { useSelection } from './selection.ts'
import { useMods } from './store.ts'
import { useListHeading } from './useListHeading.ts'
import { useModGroups } from './useModGroups.ts'
import {
  flattenModGroups,
  focusModAt,
  GROUP_HEADER_PX,
  gridColumnCount,
  gridLanePx,
  listRowId,
  orderedModIds,
  stepId,
  useModReveal,
  useModTypeahead,
  useModVirtual,
  type VirtualRow,
} from './virtualRows.ts'

const OFF_OPACITY = 0.6
const CARD_HEIGHT_SMALL = 50
const CARD_HEIGHT_MEDIUM = 64
const CARD_HEIGHT_LARGE = 80
const LARGE_LANE_EXTRA = 16
const TILE_COMPACT_PX = 38
const TILE_COMPACT_FONT_PX = 19
const CARD_GAP_PX = '10px'
const CARD_RADIUS_PX = '6px'
const NAME_FONT_PX = 14
const NAME_WEIGHT = 600
const META_FONT_PX = 12
const TAG_MAX_PX = 96
const LANE_GAP_PX = '6px'
const DIVIDER = '1px solid var(--mortar-hairline-12)'

function cardHeightPx(size: string): number {
  if (size === 'large') {
    return CARD_HEIGHT_LARGE
  }
  if (size === 'small') {
    return CARD_HEIGHT_SMALL
  }
  return CARD_HEIGHT_MEDIUM
}

const ARROWS: Record<string, string | undefined> = { ArrowLeft: 'left', ArrowRight: 'right' }

interface ModCardProps {
  mod: Mod
  orderedIds: readonly string[]
  profile: Profile
  // The column count at the time of a key press, read from a ref so a change of width re-lays the grid out without
  // re-rendering every card.
  columnsRef: { readonly current: number }
  onMove: (id: string, delta: number) => void
}

const sameCard = (a: ModCardProps, b: ModCardProps) =>
  a.mod === b.mod &&
  a.orderedIds === b.orderedIds &&
  a.onMove === b.onMove &&
  (a.profile === b.profile || ownSlice(a.profile, a.mod.key) === ownSlice(b.profile, b.mod.key))

function ModCardView({ mod: m, orderedIds, profile, columnsRef, onMove }: ModCardProps) {
  const { t } = useLingui()
  const openDetail = useDetail((s) => s.show)
  const askRemove = useMods((s) => s.askRemove)
  const id = modId(m)
  const marked = useMarked(id)
  const tabStop = useTabStop(orderedIds, id)
  const fresh = useNexusFresh(nexusIdOf(profile, m))
  const tag = firstTag(entryOf(profile, m.key)?.tags)
  const cardSize = useSettings((s) => s.gridCardSize) || 'medium'
  const showAuthor = useSettings((s) => s.showAuthorOnCards) !== false
  return (
    <Card
      {...contextMenuProps(m)}
      sx={{
        height: cardHeightPx(cardSize),
        pl: space.gap,
        pr: 0.75,
        display: 'flex',
        alignItems: 'center',
        gap: CARD_GAP_PX,
        minWidth: 0,
        borderRadius: CARD_RADIUS_PX,
        outline: marked ? '1px solid' : 'none',
        outlineColor: 'primary.main',
        // The title button's ::after covers the card, so it is the card's one click target; everything else that
        // is interactive sits above it.
        position: 'relative',
        '& > :not(.card-main)': { position: 'relative', zIndex: 1 },
        [`&:has(.card-title:focus-visible), ${PAD_FOCUS} &:has(.card-title:focus)`]: {
          outline: '2px solid',
          outlineColor: 'primary.main',
        },
        [compact]: {
          height: CARD_HEIGHT_SMALL,
          '& .tile': {
            width: TILE_COMPACT_PX,
            height: TILE_COMPACT_PX,
            fontSize: TILE_COMPACT_FONT_PX,
          },
        },
      }}
    >
      <Box
        className="card-main"
        sx={{
          flex: 1,
          minWidth: 0,
          height: '100%',
          display: 'flex',
          alignItems: 'center',
          gap: CARD_GAP_PX,
        }}
      >
        <LetterTile mod={m} fresh={fresh} />
        <Box
          sx={{
            flex: 1,
            minWidth: 0,
            pl: CARD_GAP_PX,
            borderLeft: DIVIDER,
            opacity: m.enabled ? 1 : OFF_OPACITY,
          }}
        >
          <ButtonBase
            className="card-title"
            data-mod-id={id}
            aria-label={t`Details of ${m.name}`}
            tabIndex={tabStop ? 0 : -1}
            onKeyDown={(e) => {
              const run: Partial<Record<string, () => void>> = {
                left: () => onMove(id, -1),
                right: () => onMove(id, 1),
                'mod-up': () => onMove(id, -columnsRef.current),
                'mod-down': () => onMove(id, columnsRef.current),
                'mod-toggle': () => toggleActing(m).catch(reportUnexpected),
                'mod-details': () => openDetail(m),
                'mod-remove': () => askRemove(actingMods(m)),
              }
              const action =
                run[boundShortcut(e, useSettings.getState().shortcuts) ?? ARROWS[e.key] ?? '']
              // Type-ahead has already taken a key it swallowed, such as a space inside a name being typed.
              if (action && e.target === e.currentTarget && !e.defaultPrevented) {
                e.preventDefault()
                action()
              }
            }}
            onMouseDown={(e) => {
              if (e.shiftKey) {
                e.preventDefault()
              }
            }}
            onClick={(e) => {
              useSelection.getState().click(orderedIds, id, e)
              openDetail(m)
            }}
            sx={{
              position: 'static',
              display: 'block',
              width: '100%',
              minWidth: 0,
              textAlign: 'left',
              fontFamily: 'inherit',
              color: 'inherit',
              [`&:focus-visible, ${PAD_FOCUS} &:focus`]: { outline: 'none' },
              '&::after': { content: '""', position: 'absolute', inset: 0 },
            }}
          >
            <Typography
              noWrap={true}
              title={m.name}
              sx={{ fontSize: NAME_FONT_PX, fontWeight: NAME_WEIGHT }}
            >
              {m.name}
            </Typography>
          </ButtonBase>
          <Typography
            noWrap={true}
            title={showAuthor ? `${formatAuthors(m.author)} · ${m.version}` : m.version}
            sx={{ fontSize: META_FONT_PX, color: 'text.secondary' }}
          >
            {showAuthor ? (
              <>
                <Box component="span" sx={{ position: 'relative', zIndex: 1 }}>
                  <AuthorLink authorField={m.author} mod={m} profile={profile} />
                </Box>
                {` · ${m.version}`}
              </>
            ) : (
              m.version
            )}
          </Typography>
        </Box>
      </Box>
      <ExtraFilesChip mod={m} profile={profile} />
      <OverlayCountChip mod={m} profile={profile} />
      <PinBadge mod={m} />
      <CompatChip mod={m} />
      <UpdateBadge mod={m} />
      <NexusGoneBadge mod={m} />
      <ProblemBadge mod={m} />
      <LiveBadge mod={m} />
      <LastRunBadge mod={m} />
      {tag ? <Chip size="small" label={tag} title={tag} sx={{ maxWidth: TAG_MAX_PX }} /> : null}
      <ModMenu mod={m} />
    </Card>
  )
}

const ModCard = memo(ModCardView, sameCard)

// The rows of the virtual window as one grid: every card keeps this parent when the column count changes, so opening the
// details panel (which narrows the grid) moves the cards by CSS instead of mounting them again. A group header spans the
// grid, and each row is as tall as the virtualizer sized it (a card plus the gap, a header's 36 px).
function GridWindow({
  rows,
  top,
  heading,
  collapsed,
  gameId,
  setCollapsed,
  groupBy,
  tagHint,
  columns,
  columnsRef,
  orderedIds,
  profile,
  groups,
  onMove,
}: {
  rows: readonly VirtualRow<ListRow>[]
  top: number
  heading: (key: string) => string
  collapsed: Record<string, boolean>
  gameId: string
  setCollapsed: (fn: (cur: Record<string, boolean>) => Record<string, boolean>) => void
  groupBy: string
  tagHint: string
  columns: number
  columnsRef: { readonly current: number }
  orderedIds: readonly string[]
  profile: Profile
  groups: readonly { key: string; items: readonly ListRow[] }[]
  onMove: (id: string, delta: number) => void
}) {
  return (
    <Box
      sx={{
        position: 'absolute',
        top: 0,
        left: 0,
        width: '100%',
        transform: `translateY(${top}px)`,
        display: 'grid',
        gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
        gap: LANE_GAP_PX,
        px: space.pad,
        alignContent: 'start',
      }}
    >
      {rows.flatMap((item) => {
        if (item.kind === 'header') {
          return (
            <Box
              key={item.key}
              sx={{ gridColumn: '1 / -1', height: GROUP_HEADER_PX, mx: -2, mb: `-${LANE_GAP_PX}` }}
            >
              <GroupHeaderRow
                groupKey={item.groupKey}
                count={item.count}
                label={heading(item.groupKey)}
                collapsed={collapsed}
                gameId={gameId}
                setCollapsed={setCollapsed}
                groupBy={groupBy}
                tagHint={tagHint}
                groups={groups}
              />
            </Box>
          )
        }
        if (item.kind !== 'lane') {
          return []
        }
        return item.items.map((r) => (
          <ModCard
            key={modId(r.mod)}
            mod={r.mod}
            orderedIds={orderedIds}
            profile={profile}
            columnsRef={columnsRef}
            onMove={onMove}
          />
        ))
      })}
    </Box>
  )
}

function CardsPane({
  groups,
  groupBy,
  collapsed,
  setCollapsed,
  gameId,
  heading,
  tagHint,
  orderedIds,
  profile,
}: {
  groups: { key: string; items: ListRow[] }[]
  groupBy: string
  collapsed: Record<string, boolean>
  setCollapsed: (fn: (cur: Record<string, boolean>) => Record<string, boolean>) => void
  gameId: string
  heading: (key: string) => string
  tagHint: string
  orderedIds: readonly string[]
  profile: Profile
}) {
  const compactCards = useMediaQuery(compactQuery)
  const cardSize = useSettings((s) => s.gridCardSize) || 'medium'
  const laneCompact = compactCards || cardSize === 'small'
  const lanePx = gridLanePx(laneCompact) + (cardSize === 'large' ? LARGE_LANE_EXTRA : 0)
  const [width, setWidth] = useState(0)
  const columns = gridColumnCount(width)
  const columnsRef = useRef(columns)
  columnsRef.current = columns
  const items = useMemo(
    () =>
      flattenModGroups(groups, {
        grouped: groupBy !== 'none',
        collapsed,
        idOf: (row) => modId(row.mod),
        columns,
      }),
    [collapsed, columns, groupBy, groups],
  )
  const { parentRef, virtualizer } = useModVirtual(items, lanePx)
  const virtualItems = virtualizer.getVirtualItems()
  const visible = virtualItems.flatMap((vi) => items[vi.index] ?? [])
  const visibleStart = virtualItems[0]?.start ?? 0
  const detailId = useDetail((s) => s.detailId)
  // Measured before paint, so the first frame lays out the real column count rather than one column.
  useLayoutEffect(() => {
    const el = parentRef.current
    if (!el) {
      return
    }
    const sync = () => setWidth(el.clientWidth)
    const ro = new ResizeObserver(sync)
    ro.observe(el)
    sync()
    return () => ro.disconnect()
  }, [parentRef])
  useModReveal({
    detailId,
    groups,
    items,
    idOf: listRowId,
    collapsed,
    setCollapsed,
    gameId,
    virtualizer,
  })
  const navIds = useMemo(() => orderedModIds(items, (row) => modId(row.mod)), [items])
  const move = useRef<(id: string, delta: number) => void>(() => undefined)
  move.current = (id, delta) => {
    const next = stepId(navIds, id, delta)
    if (next && next !== id) {
      focusModAt({ items, idOf: listRowId, virtualizer, parentRef }, next)
      showModId(next)
    }
  }
  // Stable, so memoised cards keep their props across renders while still stepping through the current items.
  const onMove = useCallback((id: string, delta: number) => move.current(id, delta), [])
  useModTypeahead({
    items,
    nameOf: (row) => row.mod.name,
    idOf: listRowId,
    virtualizer,
    parentRef,
  })
  return (
    <Box
      ref={parentRef}
      tabIndex={0}
      sx={{
        minHeight: 0,
        height: '100%',
        overflowY: 'auto',
        '&:focus-visible': { outlineOffset: -2 },
      }}
    >
      <Box sx={{ height: virtualizer.getTotalSize(), position: 'relative' }}>
        <GridWindow
          rows={visible}
          top={visibleStart}
          heading={heading}
          collapsed={collapsed}
          gameId={gameId}
          setCollapsed={setCollapsed}
          groupBy={groupBy}
          tagHint={tagHint}
          columns={columns}
          columnsRef={columnsRef}
          orderedIds={orderedIds}
          profile={profile}
          groups={groups}
          onMove={onMove}
        />
      </Box>
    </Box>
  )
}

export function Cards({ shown, profile }: { shown: Mod[]; profile: Profile }) {
  const { t } = useLingui()
  const { groupBy, gameId, collapsed, setCollapsed, groups, orderedIds } = useModGroups(
    shown,
    profile,
  )
  const tagHint = t`A mod with several tags appears under its first tag.`
  const heading = useListHeading(groupBy)
  return (
    <CardsPane
      groups={groups}
      groupBy={groupBy}
      collapsed={collapsed}
      setCollapsed={setCollapsed}
      gameId={gameId}
      heading={heading}
      tagHint={tagHint}
      orderedIds={orderedIds}
      profile={profile}
    />
  )
}
