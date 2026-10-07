import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { alpha, Box, TableCell, type TableCellProps, TableRow, useMediaQuery } from '@mui/material'
import { Pin } from 'lucide-react'
import { type MouseEvent, memo, type ReactNode, useMemo, useState } from 'react'
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
import { CompatChip } from './CompatChip.tsx'
import { useColumnAvailable, useSavedColumns } from './contributedColumns.ts'
import { useDetail } from './detail.ts'
import { ExtraFilesChip } from './ExtraFilesChip.tsx'
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
import { modId, nexusIdOf } from './lookup.ts'
import { ModListTable } from './ModListVirtual.tsx'
import { useMarked, useTabStop } from './marked.ts'
import { contextMenuProps } from './menu.ts'
import { useNexusFresh } from './nexusDetails.ts'
import { formatCount, isNewer } from './nexusFormat.ts'
import { OverlayCountChip } from './OverlayRow.tsx'
import { nestOverlays, overlaysByBase, ownSlice } from './overlayRows.ts'
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
      return (
        <Cell key="name" title={m.name} sx={{ ...ellipsis, fontWeight: 500 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, minWidth: 0 }}>
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
    case 'notes': {
      const notes = [row.note, row.tags.join(', ')].filter((part) => part !== '').join(' · ')
      if (id === 'updated') {
        return <ValueCell key={id} text={<When value={page?.updated ?? ''} />} />
      }
      if (id === 'installed') {
        return <ValueCell key={id} text={<When value={row.added} />} />
      }
      const text = {
        id: dash(m.id),
        author: dash(m.author),
        source: dash(row.source),
        category: dash(row.categoryLabel),
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
  striped: boolean
  cols: readonly ListColumnId[]
  locale: string
  orderedIds: readonly string[]
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
  a.striped === b.striped &&
  a.locale === b.locale &&
  a.orderedIds === b.orderedIds &&
  a.onArrow === b.onArrow &&
  a.cols.join() === b.cols.join() &&
  sameRowFields(a.row, b.row) &&
  (a.profile === b.profile ||
    ownSlice(a.profile, a.row.mod.key) === ownSlice(b.profile, b.row.mod.key))

function ModRowView({ row, striped, cols, locale, orderedIds, profile, onArrow }: ModRowProps) {
  const show = useDetail((s) => s.show)
  const askRemove = useMods((s) => s.askRemove)
  const m = row.mod
  const rowId = modId(m)
  const marked = useMarked(rowId)
  const tabStop = useTabStop(orderedIds, rowId)
  const menu = contextMenuProps(m)
  const fresh = useNexusFresh(nexusIdOf(profile, m))
  return (
    <TableRow
      role="row"
      hover={true}
      selected={marked}
      onMouseDown={(e) => {
        if (e.shiftKey) {
          e.preventDefault()
        }
      }}
      onClick={(e) => {
        useSelection.getState().click(orderedIds, rowId, e)
        show(m)
      }}
      data-mod-row="true"
      data-mod-id={rowId}
      tabIndex={tabStop ? 0 : -1}
      {...menu}
      onKeyDown={(e) => {
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
        bgcolor: (th) => {
          if (marked) {
            return alpha(th.palette.primary.main, SELECTED_ALPHA)
          }
          return striped ? 'var(--mortar-hairline-ghost)' : 'transparent'
        },
      }}
    >
      {cols.flatMap((id) =>
        id === 'name'
          ? [
              <Cell key="tile">
                <LetterTile mod={m} size={26} fresh={fresh} />
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
  const onCommit = () => {
    if (preview) {
      persistColumns([...preview, ...visible.filter((id) => !preview.includes(id))])
    }
    setPreview(null)
  }

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
        renderRow={(row, striped, onArrow) => (
          <ModRow
            row={row}
            striped={striped}
            cols={cols}
            locale={i18n.locale}
            orderedIds={orderedIds}
            profile={profile}
            onArrow={onArrow}
          />
        )}
      />
    </Box>
  )
}
