import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { alpha, Box, TableCell, type TableCellProps, TableRow, useMediaQuery } from '@mui/material'
import { CornerDownRight, Pin } from 'lucide-react'
import {
  type MouseEvent,
  memo,
  type ReactNode,
  useCallback,
  useMemo,
  useRef,
  useState,
} from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { formatDuration } from '../console/startupView.ts'
import { compactQuery } from '../game/compact.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { When } from '../i18n/When.tsx'
import { boundShortcut, type ShortcutId } from '../settings/shortcuts.ts'
import { useSettings } from '../settings/store.ts'
import { useWidth } from '../shell/useWidth.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { actingMods, toggleActing } from './actingMods.ts'
import { formatAuthors } from './authorNormalize.ts'
import { CompatChip } from './CompatChip.tsx'
import { useColumnAvailable, useSavedColumns } from './contributedColumns.ts'
import { useDetail } from './detail.ts'
import { ExtraFilesChip } from './ExtraFilesChip.tsx'
import { FavouriteStar } from './FavouriteStar.tsx'
import { entryFavouriteKey } from './favourites.ts'
import {
  columnMenuFromEvent,
  fitListColumns,
  type ListColumnId,
  type ListRow,
  listGridColumns,
  persistColumns,
  sanitizeListColumns,
  visibleListColumns,
} from './listColumns.ts'
import { entryOf, modId } from './lookup.ts'
import { ModListTable } from './ModListVirtual.tsx'
import { useMarked, useRowFocus, useTabStop } from './marked.ts'
import { contextMenuProps, openModPage } from './menu.ts'
import { useModFresh } from './modFresh.ts'
import { formatCount, isNewer } from './nexusFormat.ts'
import { OverlayCountChip } from './OverlayRow.tsx'
import { nestOverlays, overlaysByBase, ownSlice } from './overlayRows.ts'
import { MIDDLE, opensPage } from './pageClick.ts'
import {
  LastRunBadge,
  LetterTile,
  LiveBadge,
  ModSwitch,
  NexusGoneBadge,
  PinBadge,
  ProblemBadge,
  UpdateBadge,
} from './parts.tsx'
import { useSelection } from './selection.ts'
import { useMods } from './store.ts'
import { useListHeading } from './useListHeading.ts'
import { useModGroups } from './useModGroups.ts'
import { flattenModGroups } from './virtualRows.ts'

const SELECTED_ALPHA = 0.14
const ellipsis = { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } as const

const cellBase = { p: 0, border: 0, fontSize: 'inherit', color: 'inherit' } as const

function Cell({ sx, ...props }: TableCellProps) {
  return <TableCell role="cell" {...props} sx={{ ...cellBase, ...sx }} />
}

function dash(value: string) {
  return value || '—'
}

function ValueCell({ text, title, accent }: { text: ReactNode; title?: string; accent?: boolean }) {
  return (
    <Cell
      title={title ?? (typeof text === 'string' ? text : undefined)}
      sx={{
        ...ellipsis,
        color: accent ? 'primary.main' : 'text.secondary',
        fontVariantNumeric: 'tabular-nums',
      }}
    >
      {text}
    </Cell>
  )
}

function MeasureCell({
  column,
  row,
  locale,
}: {
  column: 'size' | 'startup' | 'order'
  row: ListRow
  locale: string
}) {
  if (column === 'size') {
    return <ValueCell text={row.size === undefined ? '—' : formatBytes(row.size)} />
  }
  if (column === 'order') {
    return <ValueCell text={row.order ? String(row.order) : '—'} />
  }
  return (
    <ValueCell text={row.startupMs === undefined ? '—' : formatDuration(row.startupMs, locale)} />
  )
}

function OverridesNote({ count }: { count: number }) {
  const { t } = useLingui()
  return (
    <Box
      component="span"
      sx={{ ...ellipsis, flexShrink: 0, fontSize: 12, fontWeight: 400, color: 'text.secondary' }}
    >
      {t`overrides ${plural(count, { one: '# file', other: '# files' })}`}
    </Box>
  )
}

// The name with its chips, beneath a ↳ when the list nests it under the mod that requires it.
function NameCell({ row, profile }: { row: ListRow; profile: Profile }) {
  const m = row.mod
  return (
    <Cell title={m.name} sx={{ ...ellipsis, fontWeight: 500 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, minWidth: 0 }}>
        {row.depth ? (
          <Box
            component="span"
            sx={{
              display: 'flex',
              flexShrink: 0,
              ml: (row.depth - 1) * 2,
              color: 'text.secondary',
            }}
          >
            <CornerDownRight size={14} aria-hidden={true} />
          </Box>
        ) : null}
        <Box component="span" sx={{ ...ellipsis, minWidth: 0 }}>
          {m.name}
        </Box>
        <ExtraFilesChip mod={m} profile={profile} />
        <OverlayCountChip mod={m} profile={profile} />
        <ProblemBadge mod={m} />
        {row.overrides ? <OverridesNote count={row.overrides} /> : null}
      </Box>
    </Cell>
  )
}

function cellsFor(id: ListColumnId, row: ListRow, locale: string, profile: Profile) {
  const m = row.mod
  const page = row.details?.page
  switch (id) {
    case 'on':
      return (
        <Cell key="on">
          <ModSwitch mod={m} />
        </Cell>
      )
    case 'name':
      return <NameCell key="name" row={row} profile={profile} />
    case 'version':
      return (
        <Cell key="version" title={dash(m.version)} sx={{ ...ellipsis, color: 'text.secondary' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, minWidth: 0 }}>
            <Box component="span" sx={{ ...ellipsis, fontVariantNumeric: 'tabular-nums' }}>
              {dash(m.version)}
            </Box>
            {row.pinned ? <Pin size={14} aria-hidden={true} /> : null}
          </Box>
        </Cell>
      )
    case 'latest': {
      const latest = page?.version ?? ''
      return (
        <ValueCell
          key="latest"
          text={dash(latest)}
          accent={latest !== '' && isNewer(latest, m.version)}
        />
      )
    }
    case 'id':
    case 'author':
    case 'source':
    case 'category':
    case 'updated':
    case 'installed':
    case 'switchedOff':
    case 'reason':
    case 'notes': {
      const notes = [row.note, row.tags.join(', ')].filter((part) => part !== '').join(' · ')
      if (id === 'updated') {
        return <ValueCell key={id} text={<When value={page?.updated ?? ''} />} />
      }
      if (id === 'installed') {
        return <ValueCell key={id} text={<When value={row.added} />} />
      }
      if (id === 'switchedOff') {
        return <ValueCell key={id} text={<When value={row.switchedOff} />} />
      }
      const text = {
        id: dash(m.id),
        author: dash(formatAuthors(m.author)),
        source: dash(row.source),
        category: dash(row.categoryLabel),
        reason: row.reason,
        notes: dash(notes),
      }[id]
      return <ValueCell key={id} text={text} />
    }
    case 'endorsements': {
      const n = page?.endorsements ?? m.endorsements
      const missing = page === undefined && m.endorsements === 0
      return <ValueCell key="endorsements" text={missing ? '—' : formatCount(n, locale)} />
    }
    case 'downloads':
      return (
        <ValueCell
          key="downloads"
          text={page === undefined ? '—' : formatCount(page.downloads, locale)}
        />
      )
    case 'needs': {
      const names = m.needs
      if (names === undefined || names === null) {
        return <ValueCell key="needs" text="—" />
      }
      const text = names.length === 0 ? '0' : names.join(', ')
      return <ValueCell key="needs" text={text} />
    }
    case 'status':
      return (
        <Cell
          key="status"
          sx={{
            fontSize: 13,
            whiteSpace: 'nowrap',
            color: m.enabled ? 'text.primary' : 'text.secondary',
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, minWidth: 0 }}>
            <PinBadge mod={m} />
            <CompatChip mod={m} />
            <LiveBadge mod={m} />
            <NexusGoneBadge mod={m} />
            <UpdateBadge mod={m} />
            <Box component="span" title={row.status} sx={ellipsis}>
              {row.status}
            </Box>
          </Box>
        </Cell>
      )
    case 'lastRun':
      return (
        <Cell key="lastRun">
          <LastRunBadge mod={m} />
        </Cell>
      )
    case 'size':
    case 'startup':
    case 'order':
      return <MeasureCell key={id} column={id} row={row} locale={locale} />
    default:
      return null
  }
}

interface ModRowProps {
  row: ListRow
  cols: readonly ListColumnId[]
  locale: string
  // The order, for the first row only (see useTabStop).
  first: readonly string[] | null
  // The current order for a shift-click range, read from a ref so a mod moving group re-renders only its own row.
  orderedRef: { readonly current: readonly string[] }
  profile: Profile
  onArrow: (id: string, dir: -1 | 1) => void
}

// List rows are rebuilt whenever the mods or the profile change, so a row compares their fields, not their identity.
function sameRowFields(a: ListRow, b: ListRow) {
  const keys = Object.keys(a) as (keyof ListRow)[]
  return (
    keys.length === Object.keys(b).length &&
    keys.every((k) => (k === 'tags' ? a.tags.join('\0') === b.tags.join('\0') : a[k] === b[k]))
  )
}

const sameRow = (a: ModRowProps, b: ModRowProps) =>
  a.locale === b.locale &&
  a.first === b.first &&
  a.orderedRef === b.orderedRef &&
  a.onArrow === b.onArrow &&
  a.cols.join() === b.cols.join() &&
  sameRowFields(a.row, b.row) &&
  (a.profile === b.profile ||
    ownSlice(a.profile, a.row.mod.key) === ownSlice(b.profile, b.row.mod.key))

function ModRowView({ row, cols, locale, first, orderedRef, profile, onArrow }: ModRowProps) {
  const show = useDetail((s) => s.show)
  const askRemove = useMods((s) => s.askRemove)
  const m = row.mod
  const rowId = modId(m)
  const marked = useMarked(rowId)
  const tabStop = useTabStop(rowId, first)
  const menu = contextMenuProps(m)
  const fresh = useModFresh(profile, m)
  return (
    <TableRow
      role="row"
      hover={true}
      selected={marked}
      onMouseDown={(e) => {
        if (e.shiftKey || e.button === MIDDLE) {
          e.preventDefault()
        }
      }}
      onAuxClick={(e) => {
        if (opensPage(e, false)) {
          openModPage(m)
        }
      }}
      onClick={(e) => {
        useSelection.getState().click(orderedRef.current, rowId, e)
        show(m)
      }}
      data-mod-row="true"
      data-mod-id={rowId}
      tabIndex={tabStop ? 0 : -1}
      {...menu}
      onKeyDown={(e) => {
        // Type-ahead has already taken a key it swallowed, such as a space inside a name being typed.
        if (e.defaultPrevented) {
          return
        }
        const run: Partial<Record<ShortcutId, () => void>> = {
          'mod-up': () => onArrow(rowId, -1),
          'mod-down': () => onArrow(rowId, 1),
          'mod-toggle': () => toggleActing(m).catch(reportUnexpected),
          'mod-details': () => show(m),
          'mod-remove': () => askRemove(actingMods(m)),
        }
        const action = run[boundShortcut(e, useSettings.getState().shortcuts) ?? 'dismiss']
        if (action) {
          e.preventDefault()
          action()
          return
        }
        menu.onKeyDown(e)
      }}
      sx={{
        display: 'grid',
        gridTemplateColumns: listGridColumns(cols),
        gap: space.gap,
        alignItems: 'center',
        px: space.gutter,
        height: space.row,
        fontSize: 14,
        cursor: 'pointer',
        bgcolor: (th) => (marked ? alpha(th.palette.primary.main, SELECTED_ALPHA) : 'transparent'),
        ...(marked ? {} : { '[data-striped] > &': { bgcolor: 'var(--mortar-hairline-ghost)' } }),
      }}
    >
      {cols.flatMap((id) =>
        id === 'name'
          ? [
              <Cell key="tile">
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  <FavouriteStar starKey={entryFavouriteKey(entryOf(profile, m.key))} />
                  <LetterTile mod={m} size={26} fresh={fresh} />
                </Box>
              </Cell>,
              cellsFor(id, row, locale, profile),
            ]
          : [cellsFor(id, row, locale, profile)],
      )}
    </TableRow>
  )
}

const ModRow = memo(ModRowView, sameRow)

export function ModList({ profile, mods }: { profile: Profile; mods: Mod[] }) {
  const { t, i18n } = useLingui()
  const { groupBy, sort, gameId, collapsed, setCollapsed, groups, orderedIds } = useModGroups(
    mods,
    profile,
  )
  const { orderedRef } = useRowFocus(orderedIds)
  const narrow = useMediaQuery(compactQuery)
  const listColumns = useSavedColumns()
  const available = useColumnAvailable()
  const visible = sanitizeListColumns(listColumns).filter(available)
  const [box, width] = useWidth<HTMLDivElement>()
  const settled = fitListColumns(visibleListColumns(listColumns, narrow, available), width)
  // While a header is dragged the table shows this order, so every row moves with it.
  const [preview, setPreview] = useState<ListColumnId[] | null>(null)
  const cols = preview ?? settled
  const [menu, setMenu] = useState<{ top: number; left: number } | null>(null)
  const tagHint = t`A mod with several tags appears under its first tag.`
  const items = useMemo(
    () =>
      nestOverlays(
        flattenModGroups(groups, {
          grouped: groupBy !== 'none',
          nested: groupBy === 'group',
          collapsed,
          idOf: (row) => modId(row.mod),
        }),
        (row) => row.mod.key,
        overlaysByBase(profile),
      ),
    [collapsed, groupBy, groups, profile],
  )
  const onMenu = (e: MouseEvent) => setMenu(columnMenuFromEvent(e))
  const grid = listGridColumns(cols)
  const headingFor = useListHeading(groupBy)
  // The memoised header keeps the handler it first got while its columns stay put; reading the latest from a ref
  // commits against the current columns even when one was added that the width hides.
  const commit = useRef<() => void>(() => undefined)
  commit.current = () => {
    if (preview) {
      persistColumns([...preview, ...visible.filter((id) => !preview.includes(id))])
    }
    setPreview(null)
  }
  const onCommit = useCallback(() => commit.current(), [])

  return (
    <Box ref={box} sx={{ minWidth: 0, minHeight: 0, height: '100%' }}>
      <ModListTable
        grid={grid}
        cols={cols}
        sort={sort}
        onMenu={onMenu}
        onPreview={setPreview}
        onCommit={onCommit}
        onCancel={() => setPreview(null)}
        groups={groups}
        items={items}
        groupBy={groupBy}
        headingFor={headingFor}
        tagHint={tagHint}
        collapsed={collapsed}
        gameId={gameId}
        setCollapsed={setCollapsed}
        visible={visible}
        menu={menu}
        setMenu={setMenu}
        renderRow={(row, onArrow) => (
          <ModRow
            row={row}
            cols={cols}
            locale={i18n.locale}
            first={orderedIds[0] === modId(row.mod) ? orderedIds : null}
            orderedRef={orderedRef}
            profile={profile}
            onArrow={onArrow}
          />
        )}
      />
    </Box>
  )
}
