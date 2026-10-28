import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { alpha, Box, TableRow, useMediaQuery } from '@mui/material'
import { CornerDownRight, Pin } from 'lucide-react'
import {
  type ComponentProps,
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
import { openProfileOf, useProfiles } from '../profiles/store.ts'
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
import { LetterTile } from './LetterTile.tsx'
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
import { figureNumber, figureText } from './listFigures.ts'
import { entryOf, modId, useLocalMark } from './lookup.ts'
import { ModListTable } from './ModListVirtual.tsx'
import { useMarked, useRowFocus, useTabStop } from './marked.ts'
import { contextMenuProps } from './menu.ts'
import { useModFresh } from './modFresh.ts'
import { cellsSx } from './modListCells.ts'
import { formatCount, isNewer } from './nexusFormat.ts'
import { OverlayCountChip } from './OverlayRow.tsx'
import { cardSlice, nestOverlays, overlaysByBase } from './overlayRows.ts'
import { PackageKind } from './PackageKind.tsx'
import {
  LastRunBadge,
  LiveBadge,
  ModSwitch,
  NexusGoneBadge,
  PinBadge,
  ProblemBadge,
  UpdateBadge,
} from './parts.tsx'
import { rowPointerHandlers } from './rowPointer.ts'
import { useMods } from './store.ts'
import { useListHeading } from './useListHeading.ts'
import { useModGroups } from './useModGroups.ts'
import { flattenModGroups } from './virtualRows.ts'

const SELECTED_ALPHA = 0.14
// How far each nesting level indents an implicit mod's name: the theme's 2 spacing units.
const DEPTH_PX = 16

function Cell(props: ComponentProps<'td'>) {
  return <td {...props} />
}

function dash(value: string) {
  return value || '—'
}

function ValueCell({ text, title, accent }: { text: ReactNode; title?: string; accent?: boolean }) {
  return (
    <Cell
      title={title ?? (typeof text === 'string' ? text : undefined)}
      className={accent ? 'cell-value cell-accent' : 'cell-value'}
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

// The columns whose text comes from the mod's own files or its source (listFigures).
function FigureCell({
  column,
  row,
  locale,
}: {
  column:
    | 'latest'
    | 'released'
    | 'updated'
    | 'pluginVersion'
    | 'kind'
    | 'id'
    | 'endorsements'
    | 'downloads'
  row: ListRow
  locale: string
}) {
  if (column === 'endorsements' || column === 'downloads') {
    const n = figureNumber(row, column)
    return <ValueCell text={n === undefined ? '—' : formatCount(n, locale)} />
  }
  const text = figureText(row, column)
  if (column === 'released' || column === 'updated') {
    return <ValueCell text={<When value={text} />} />
  }
  if (column === 'kind') {
    return <ValueCell text={<PackageKind kind={text} />} />
  }
  return (
    <ValueCell
      text={dash(text)}
      accent={column === 'latest' && text !== '' && isNewer(text, row.mod.version)}
    />
  )
}

function OverridesNote({ count }: { count: number }) {
  const { t } = useLingui()
  return (
    <span className="cell-overrides">
      {t`overrides ${plural(count, { one: '# file', other: '# files' })}`}
    </span>
  )
}

// The pin beside a version, read on its own: freezing a profile pins every mod, and only this mark changes.
function VersionPin({ mod }: { mod: Mod }) {
  const pinned = useProfiles((s) => entryOf(openProfileOf(s), mod.key)?.pinned === true)
  return pinned ? <Pin size={14} aria-hidden={true} /> : null
}

// The name with its chips, beneath a ↳ when the list nests it under the mod that requires it.
function NameCell({ row, profile }: { row: ListRow; profile: Profile }) {
  const m = row.mod
  return (
    <Cell title={m.name} className="cell-name">
      <span className="cell-line">
        {row.depth ? (
          <span className="cell-depth" style={{ marginLeft: (row.depth - 1) * DEPTH_PX }}>
            <CornerDownRight size={14} aria-hidden={true} />
          </span>
        ) : null}
        <span className="cell-text">{m.name}</span>
        <ExtraFilesChip mod={m} profile={profile} />
        <OverlayCountChip mod={m} profile={profile} />
        <ProblemBadge mod={m} />
        {row.overrides ? <OverridesNote count={row.overrides} /> : null}
      </span>
    </Cell>
  )
}

function cellsFor(id: ListColumnId, row: ListRow, locale: string, profile: Profile) {
  const m = row.mod
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
        <Cell key="version" title={dash(m.version)} className="cell-version">
          <span className="cell-line">
            <span className="cell-text">{dash(m.version)}</span>
            <VersionPin mod={m} />
          </span>
        </Cell>
      )
    case 'latest':
    case 'released':
    case 'updated':
    case 'pluginVersion':
    case 'kind':
    case 'id':
    case 'endorsements':
    case 'downloads':
      return <FigureCell key={id} column={id} row={row} locale={locale} />
    case 'author':
    case 'source':
    case 'category':
    case 'installed':
    case 'switchedOff':
    case 'reason':
    case 'notes': {
      const notes = [row.note, row.tags.join(', ')].filter((part) => part !== '').join(' · ')
      if (id === 'installed') {
        return <ValueCell key={id} text={<When value={row.added} />} />
      }
      if (id === 'switchedOff') {
        return <ValueCell key={id} text={<When value={row.switchedOff} />} />
      }
      const text = {
        author: dash(formatAuthors(m.author)),
        source: dash(row.source),
        category: dash(row.categoryLabel),
        reason: row.reason,
        notes: dash(notes),
      }[id]
      return <ValueCell key={id} text={text} />
    }
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
        <Cell key="status" className={m.enabled ? 'cell-status' : 'cell-status cell-dim'}>
          <span className="cell-line">
            <PinBadge mod={m} />
            <CompatChip mod={m} />
            <LiveBadge mod={m} />
            <NexusGoneBadge mod={m} />
            <UpdateBadge mod={m} />
            <span title={row.status} className="cell-text">
              {row.status}
            </span>
          </span>
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

// List rows are rebuilt whenever the mods, the profile or the problems change, so a row compares their fields, not
// their identity; tags and figures are built fresh each time and compare by content.
function sameRowFields(a: ListRow, b: ListRow) {
  const keys = Object.keys(a) as (keyof ListRow)[]
  return keys.length === Object.keys(b).length && keys.every((k) => sameField(k, a, b))
}

function sameField(k: keyof ListRow, a: ListRow, b: ListRow) {
  // The pin shows through VersionPin and PinBadge, which read the store themselves.
  if (k === 'pinned') {
    return true
  }
  if (k === 'tags') {
    return a.tags.join('\0') === b.tags.join('\0')
  }
  if (k === 'figures' && a.figures && b.figures) {
    const [x, y] = [a.figures, b.figures]
    return (
      x.latest === y.latest &&
      x.released === y.released &&
      x.updated === y.updated &&
      x.rating === y.rating &&
      x.downloads === y.downloads
    )
  }
  return a[k] === b[k]
}

type CellProps = Pick<ModRowProps, 'locale' | 'cols' | 'row' | 'profile'>

// What a row's cells show: a profile change elsewhere leaves the row's own slice alone.
const sameCells = (a: CellProps, b: CellProps) =>
  a.locale === b.locale &&
  a.cols.join() === b.cols.join() &&
  sameRowFields(a.row, b.row) &&
  (a.profile === b.profile ||
    cardSlice(a.profile, a.row.mod.key) === cardSlice(b.profile, b.row.mod.key))

const sameRow = (a: ModRowProps, b: ModRowProps) =>
  a.first === b.first && a.orderedRef === b.orderedRef && a.onArrow === b.onArrow && sameCells(a, b)

function ModRowView({ row, cols, locale, first, orderedRef, profile, onArrow }: ModRowProps) {
  const show = useDetail((s) => s.show)
  const askRemove = useMods((s) => s.askRemove)
  const m = row.mod
  const rowId = modId(m)
  const marked = useMarked(rowId)
  const tabStop = useTabStop(rowId, first)
  const menu = contextMenuProps(m)
  const fresh = useModFresh(profile, m)
  const local = useLocalMark(profile, m)
  return (
    <TableRow
      role="row"
      hover={true}
      selected={marked}
      {...rowPointerHandlers(m, rowId, orderedRef, show)}
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
      sx={[
        cellsSx,
        {
          display: 'grid',
          gridTemplateColumns: listGridColumns(cols),
          gap: space.gap,
          alignItems: 'center',
          px: space.gutter,
          height: space.row,
          fontSize: 'medium.fontSize',
          cursor: 'pointer',
          bgcolor: (th) =>
            marked ? alpha(th.palette.primary.main, SELECTED_ALPHA) : 'transparent',
          ...(marked ? {} : { '[data-striped] > &': { bgcolor: 'var(--mortar-hairline-ghost)' } }),
          ...(local ? { boxShadow: (th) => `inset 3px 0 0 ${th.palette.info.main}` } : {}),
        },
      ]}
    >
      <RowCells row={row} cols={cols} locale={locale} profile={profile} fresh={fresh} />
    </TableRow>
  )
}

interface RowCellsProps {
  row: ListRow
  cols: readonly ListColumnId[]
  locale: string
  profile: Profile
  fresh: boolean
}

function RowCellsView({ row, cols, locale, profile, fresh }: RowCellsProps) {
  const m = row.mod
  return cols.flatMap((id) =>
    id === 'name'
      ? [
          <Cell key="tile">
            <span className="cell-line cell-tile">
              <FavouriteStar starKey={entryFavouriteKey(entryOf(profile, m.key))} />
              <LetterTile mod={m} size={26} fresh={fresh} />
            </span>
          </Cell>,
          cellsFor(id, row, locale, profile),
        ]
      : [cellsFor(id, row, locale, profile)],
  )
}

// Selecting or focusing a row changes only the row's own highlight, so its cells skip that render.
const RowCells = memo(RowCellsView, (a, b) => a.fresh === b.fresh && sameCells(a, b))

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
