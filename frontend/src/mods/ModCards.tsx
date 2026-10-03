import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Card, Chip, Typography, useMediaQuery } from '@mui/material'
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { compact, compactQuery } from '../game/compact.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { AuthorLink } from './AuthorLink.tsx'
import { CompatChip } from './CompatChip.tsx'
import { useCustomCategories } from './customCategories.ts'
import { useDetail } from './detail.ts'
import { ExtraFilesChip } from './ExtraFilesChip.tsx'
import {
  customCategoryById,
  emptyGroupLabel,
  firstTag,
  groupHeading,
  groupSorted,
  installedNames,
  loadCollapsed,
  rowGroupKey,
  sanitizeListGroupBy,
  toggleCollapsed,
} from './group.ts'
import { compareListRows, type ListRow, sanitizeListSort } from './listColumns.ts'
import { toListRow } from './listRows.ts'
import { entryOf, modId, modStatusProblem, nexusIdOf, updateFor } from './lookup.ts'
import { ModMenu } from './ModMenu.tsx'
import { ModsGroupHeader } from './ModsGroupHeader.tsx'
import { contextMenuProps } from './menu.ts'
import { primeDetails, useNexusDetails, useNexusFresh } from './nexusDetails.ts'
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
import { setGroupEnabled } from './storeEntries.ts'
import { useUpdates } from './updates.ts'
import {
  flattenModGroups,
  gridColumnCount,
  gridLanePx,
  groupKeyHolding,
  useModTypeahead,
  useModVirtual,
  type VirtualRow,
  virtualIndexOf,
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

function ModCard({
  mod: m,
  orderedIds,
  profile,
}: {
  mod: Mod
  orderedIds: readonly string[]
  profile: Profile
}) {
  const { t } = useLingui()
  const openDetail = useDetail((s) => s.show)
  const selectedId = useDetail((s) => s.detailId)
  const selectedIds = useSelection((s) => s.ids)
  const id = modId(m)
  const marked = selectedIds.includes(id) || (selectedIds.length === 0 && id === selectedId)
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
      <ButtonBase
        data-mod-id={id}
        aria-label={t`Details of ${m.name}`}
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
          '&.Mui-focusVisible': { outlineOffset: '-2px' },
          flex: 1,
          minWidth: 0,
          height: '100%',
          display: 'flex',
          alignItems: 'center',
          gap: CARD_GAP_PX,
          justifyContent: 'flex-start',
          textAlign: 'left',
          fontFamily: 'inherit',
          color: 'inherit',
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
          <Typography
            noWrap={true}
            title={m.name}
            sx={{ fontSize: NAME_FONT_PX, fontWeight: NAME_WEIGHT }}
          >
            {m.name}
          </Typography>
          <Typography
            noWrap={true}
            title={showAuthor ? `${m.author} · ${m.version}` : m.version}
            sx={{ fontSize: META_FONT_PX, color: 'text.secondary' }}
          >
            {showAuthor ? (
              <>
                <AuthorLink authorField={m.author} mod={m} profile={profile} />
                {` · ${m.version}`}
              </>
            ) : (
              m.version
            )}
          </Typography>
        </Box>
      </ButtonBase>
      <ExtraFilesChip mod={m} profile={profile} />
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
}) {
  if (item.kind === 'header') {
    return (
      <ModsGroupHeader
        label={heading(item.groupKey)}
        count={item.count}
        open={collapsed[item.groupKey] !== true}
        onToggle={() =>
          setCollapsed((cur) =>
            toggleCollapsed(gameId, cur, item.groupKey, collapsed[item.groupKey] !== true),
          )
        }
        {...(groupBy === 'tag' ? { hint: tagHint } : {})}
        {...(groupBy === 'group' && item.groupKey !== ''
          ? {
              enabled:
                groups.find((g) => g.key === item.groupKey)?.items.every((r) => r.mod.enabled) ===
                true,
              onEnabled: (on: boolean) => {
                setGroupEnabled(item.groupKey, on)
                  .then(() => useMods.getState().load())
                  .catch(reportUnexpected)
              },
            }
          : {})}
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
        <ModCard key={modId(r.mod)} mod={r.mod} orderedIds={orderedIds} profile={profile} />
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
  const lastReveal = useRef('')
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
  useLayoutEffect(() => {
    if (!detailId) {
      return
    }
    const held = groupKeyHolding(groups, (row) => modId(row.mod) === detailId)
    if (held !== undefined && collapsed[held] === true) {
      setCollapsed((cur) => toggleCollapsed(gameId, cur, held, false))
      return
    }
    const idx = virtualIndexOf(items, detailId, (row) => modId(row.mod))
    const token = `${detailId}:${idx}`
    if (lastReveal.current === token || idx < 0) {
      return
    }
    lastReveal.current = token
    virtualizer.scrollToIndex(idx, { align: 'auto' })
  }, [collapsed, detailId, gameId, groups, items, setCollapsed, virtualizer])
  useModTypeahead({
    items,
    nameOf: (row) => row.mod.name,
    idOf: (row) => modId(row.mod),
    virtualizer,
    parentRef,
  })
  return (
    <Box
      ref={parentRef}
      tabIndex={0}
      sx={{ minHeight: 0, height: '100%', overflowY: 'auto', outline: 'none' }}
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
  const groupBy = sanitizeListGroupBy(useSettings((s) => s.listGroupBy))
  const listSortColumn = useSettings((s) => s.listSortColumn)
  const listSortDir = useSettings((s) => s.listSortDir)
  const sort = sanitizeListSort(listSortColumn ?? '', listSortDir ?? '')
  const byId = useNexusDetails((s) => s.byId)
  const customCategories = useCustomCategories((s) => s.categories)
  const customById = customCategoryById(customCategories)
  const gameId = useProfiles((s) => s.game?.id) ?? ''
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>(() => loadCollapsed(gameId))
  const problems = useMods((s) => s.problems)
  const updates = useUpdates((s) => s.updates)
  const tagHint = t`A mod with several tags appears under its first tag.`
  useEffect(() => {
    setCollapsed(loadCollapsed(gameId))
  }, [gameId])
  useEffect(() => {
    primeDetails(shown.map((m) => nexusIdOf(profile, m)).filter((id) => id > 0)).catch(
      reportUnexpected,
    )
  }, [shown, profile])
  const names = installedNames(shown)
  const groups = groupSorted(
    shown.map((m) => toListRow(m, profile, byId, customCategories)),
    groupBy,
    (row) =>
      rowGroupKey(groupBy, row, {
        hasProblem: modStatusProblem(problems, row.mod),
        hasUpdate: Boolean(updateFor(updates, row.mod, profile)),
        names,
        customById,
      }),
    (a, b) => compareListRows(a, b, sort),
  )
  const orderedIds = groups.flatMap((g) => g.items.map((r) => modId(r.mod)))
  const emptyLabel = emptyGroupLabel(groupBy, {
    category: t`Uncategorised`,
    source: t`Unknown source`,
    tag: t`Untagged`,
    author: t`Unknown author`,
    group: t`Ungrouped`,
  })
  const heading = (key: string) =>
    groupHeading(groupBy, key, {
      empty: emptyLabel,
      problems: t`Problems`,
      update: t`Update available`,
      enabled: t`Enabled`,
      disabled: t`Disabled`,
      smapi: t`SMAPI mods`,
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
