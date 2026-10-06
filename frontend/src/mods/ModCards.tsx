import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Card, Chip, Typography, useMediaQuery } from '@mui/material'
import { useEffect, useMemo, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { compact, compactQuery } from '../game/compact.ts'
import { useProfileLoader } from '../profiles/store.ts'
import { boundShortcut } from '../settings/shortcuts.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { AuthorLink } from './AuthorLink.tsx'
import { actingMods, toggleActing } from './actingMods.ts'
import { CompatChip } from './CompatChip.tsx'
import { showModId, useDetail } from './detail.ts'
import { ExtraFilesChip } from './ExtraFilesChip.tsx'
import { firstTag, listHeadingFor } from './group.ts'
import type { ListRow } from './listColumns.ts'
import { entryOf, modId, nexusIdOf } from './lookup.ts'
import { ModMenu } from './ModMenu.tsx'
import { GroupHeaderRow } from './ModsGroupHeader.tsx'
import { useMarked, useTabStop } from './marked.ts'
import { contextMenuProps } from './menu.ts'
import { useNexusFresh } from './nexusDetails.ts'
import { OverlayCountChip } from './OverlayRow.tsx'
import {
  LastRunBadge,
  LetterTile,
  NexusGoneBadge,
  PinBadge,
  ProblemBadge,
  UpdateBadge,
} from './parts.tsx'
import { useSelection } from './selection.ts'
import { useMods } from './store.ts'
import { useModGroups } from './useModGroups.ts'
import {
  flattenModGroups,
  focusModAt,
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

function ModCard({
  mod: m,
  orderedIds,
  profile,
  columns,
  onMove,
}: {
  mod: Mod
  orderedIds: readonly string[]
  profile: Profile
  columns: number
  onMove: (id: string, delta: number) => void
}) {
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
        pl: 1,
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
        '&:has(.card-title:focus-visible)': { outline: '2px solid', outlineColor: 'primary.main' },
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
                'mod-up': () => onMove(id, -columns),
                'mod-down': () => onMove(id, columns),
                'mod-toggle': () => toggleActing(m).catch(reportUnexpected),
                'mod-details': () => openDetail(m),
                'mod-remove': () => askRemove(actingMods(m)),
              }
              const action =
                run[boundShortcut(e, useSettings.getState().shortcuts) ?? ARROWS[e.key] ?? '']
              if (action && e.target === e.currentTarget) {
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
              '&:focus-visible': { outline: 'none' },
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
            title={showAuthor ? `${m.author} · ${m.version}` : m.version}
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
      <LastRunBadge mod={m} />
      {tag ? <Chip size="small" label={tag} title={tag} sx={{ maxWidth: TAG_MAX_PX }} /> : null}
      <ModMenu mod={m} />
    </Card>
  )
}

function GridSlot({
  item,
  heading,
  collapsed,
  gameId,
  setCollapsed,
  groupBy,
  tagHint,
  columns,
  orderedIds,
  profile,
  groups,
  onMove,
}: {
  item: VirtualRow<ListRow>
  heading: (key: string) => string
  collapsed: Record<string, boolean>
  gameId: string
  setCollapsed: (fn: (cur: Record<string, boolean>) => Record<string, boolean>) => void
  groupBy: string
  tagHint: string
  columns: number
  orderedIds: readonly string[]
  profile: Profile
  groups: readonly { key: string; items: readonly ListRow[] }[]
  onMove: (id: string, delta: number) => void
}) {
  if (item.kind === 'header') {
    return (
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
    )
  }
  if (item.kind !== 'lane') {
    return null
  }
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
        gap: LANE_GAP_PX,
        px: 2,
        alignContent: 'start',
      }}
    >
      {item.items.map((r) => (
        <ModCard
          key={modId(r.mod)}
          mod={r.mod}
          orderedIds={orderedIds}
          profile={profile}
          columns={columns}
          onMove={onMove}
        />
      ))}
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
  const detailId = useDetail((s) => s.detailId)
  useEffect(() => {
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
  const onMove = (id: string, delta: number) => {
    const next = stepId(navIds, id, delta)
    if (next && next !== id) {
      focusModAt({ items, idOf: listRowId, virtualizer, parentRef }, next)
      showModId(next)
    }
  }
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
        {virtualizer.getVirtualItems().map((vi) => {
          const item = items[vi.index]
          if (!item) {
            return null
          }
          return (
            <Box
              key={item.key}
              data-index={vi.index}
              sx={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                transform: `translateY(${vi.start}px)`,
              }}
            >
              <GridSlot
                item={item}
                heading={heading}
                collapsed={collapsed}
                gameId={gameId}
                setCollapsed={setCollapsed}
                groupBy={groupBy}
                tagHint={tagHint}
                columns={columns}
                orderedIds={orderedIds}
                profile={profile}
                groups={groups}
                onMove={onMove}
              />
            </Box>
          )
        })}
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
  const loaderName = useProfileLoader()?.name ?? ''
  const heading = listHeadingFor(groupBy, {
    category: t`Uncategorised`,
    source: t`Unknown source`,
    tag: t`Untagged`,
    author: t`Unknown author`,
    group: t`Ungrouped`,
    problems: t`Mods with problems`,
    update: t`Update available`,
    enabled: t`Enabled`,
    disabled: t`Off`,
    smapi: t`${loaderName} mods`,
  })
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
