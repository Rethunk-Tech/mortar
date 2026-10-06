import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Collapse,
  IconButton,
  Link,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableSortLabel,
  Tooltip,
} from '@mui/material'
import { ChevronDown, ChevronRight } from 'lucide-react'
import type {
  StartupMod,
  StartupReport,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { cmpText } from '../mods/cmpText.ts'
import { openModHandlers } from '../mods/openMod.ts'
import { useStoredState } from '../shell/useStoredState.ts'
import { useDuration, useWhy } from './startupHooks.ts'
import { foldMods, modTotal, rowAnchor, slowestEvent, whyOf } from './startupView.ts'

const PACKS_SHOWN = 25
const BRIDGE_ID = 'Rethunk.MortarSmapiBridge'
const NAME_INDENT = 4.5
// Mod, Total and Why; the Change column joins them while comparing.
const PLAIN_COLUMNS = 3

type ModSort = 'name' | 'total' | 'why' | 'change'

interface StoredSort {
  column: ModSort
  direction: 'asc' | 'desc'
}

const isStoredSort = (value: unknown): value is StoredSort | null =>
  value === null ||
  (typeof value === 'object' &&
    'column' in value &&
    'direction' in value &&
    (value.direction === 'asc' || value.direction === 'desc') &&
    (value.column === 'name' || value.column === 'total' || value.column === 'why'))

// A mod's time in this start minus its time in the one before; a mod the earlier start lacked counts all of it.
function changeOf(mod: StartupMod, previous: StartupReport | undefined): number {
  const old = (previous?.mods ?? []).find((m) => m.id === mod.id)
  return modTotal(mod) - (old ? modTotal(old) : 0)
}

function sortMods(
  mods: StartupMod[],
  column: ModSort,
  direction: 'asc' | 'desc',
  previous: StartupReport | undefined,
): StartupMod[] {
  const sign = direction === 'asc' ? 1 : -1
  const value = (mod: StartupMod) => {
    switch (column) {
      case 'total':
        return modTotal(mod)
      case 'why':
        return whyOf(mod)?.ms ?? 0
      default:
        return changeOf(mod, previous)
    }
  }
  return [...mods].sort((a, b) =>
    column === 'name' ? sign * cmpText(a.name, b.name) : sign * (value(a) - value(b)),
  )
}

// Clicking the sorted column flips it; another column starts descending, or A to Z for names.
function nextSort(active: StoredSort, column: ModSort): StoredSort {
  if (active.column === column) {
    return { column, direction: active.direction === 'desc' ? 'asc' : 'desc' }
  }
  return { column, direction: column === 'name' ? 'asc' : 'desc' }
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <>
      <Box component="span" sx={{ color: 'text.secondary' }}>
        {label}
      </Box>
      <span>{value}</span>
    </>
  )
}

function ModDetails({ mod, sampled }: { mod: StartupMod; sampled: boolean }) {
  const { t } = useLingui()
  const duration = useDuration()
  const event = slowestEvent(mod)
  const packs = mod.packs ?? []
  const assets = mod.assetMs + mod.loadMs
  return (
    <Box sx={{ py: 1, pl: 3, display: 'flex', flexDirection: 'column', gap: 1, fontSize: 13 }}>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'max-content max-content', columnGap: 3 }}>
        <Detail label={t`Entry`} value={mod.entryMs > 0 ? duration(mod.entryMs) : '—'} />
        <Detail
          label={t`Slowest event`}
          value={event ? `${duration(event[1])} · ${event[0]}` : '—'}
        />
        <Detail label={t`Assets and packs`} value={assets > 0 ? duration(assets) : '—'} />
        {sampled ? (
          <Detail label={t`Sampled`} value={mod.sampleMs ? duration(mod.sampleMs) : '—'} />
        ) : null}
      </Box>
      {packs.length > 0 ? (
        <Box sx={{ display: 'grid', gridTemplateColumns: '1fr auto', columnGap: 3 }}>
          {packs.slice(0, PACKS_SHOWN).map((p) => (
            <Box key={p.id} sx={{ display: 'contents' }}>
              <span>{p.name}</span>
              <Box component="span" sx={{ textAlign: 'right', color: 'text.secondary' }}>
                {duration(p.ms)}
              </Box>
            </Box>
          ))}
          {packs.length > PACKS_SHOWN ? (
            <Box sx={{ gridColumn: '1 / -1', color: 'text.secondary' }}>
              {plural(packs.length - PACKS_SHOWN, { one: '# more pack', other: '# more packs' })}
            </Box>
          ) : null}
        </Box>
      ) : null}
    </Box>
  )
}

function ModRow({
  mod,
  sampled,
  sort,
  open,
  onToggle,
  change,
}: {
  mod: StartupMod
  sampled: boolean
  sort: ModSort
  open: boolean
  onToggle: () => void
  change: number | null
}) {
  const { t } = useLingui()
  const duration = useDuration()
  const why = useWhy()
  const strong = (column: ModSort) => (sort === column ? { fontWeight: 600 } : undefined)
  return (
    <>
      <TableRow id={rowAnchor(mod.id)} hover={true} onClick={onToggle} sx={{ cursor: 'pointer' }}>
        <TableCell sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <IconButton
            size="small"
            aria-expanded={open}
            aria-label={open ? t`Hide details` : t`Show details`}
            onClick={(e) => {
              e.stopPropagation()
              onToggle()
            }}
            sx={{ p: 0, borderRadius: '4px', color: 'inherit' }}
          >
            {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          </IconButton>
          <Link
            component="button"
            underline="hover"
            color="inherit"
            title={t`Show in Mods`}
            {...openModHandlers({ id: mod.id })}
            sx={{
              textAlign: 'left',
              font: 'inherit',
              textDecoration: 'none',
              '&:hover': { textDecoration: 'underline' },
            }}
          >
            {mod.name}
          </Link>
          {mod.id === BRIDGE_ID ? (
            <Tooltip title={t`Mortar's own SMAPI mod; it records these timings`}>
              <Box component="span" sx={{ color: 'text.secondary', fontSize: 12, ml: 0.5 }}>
                {t`Mortar`}
              </Box>
            </Tooltip>
          ) : null}
        </TableCell>
        <TableCell align="right" sx={strong('total')}>
          {duration(modTotal(mod))}
        </TableCell>
        {change === null ? null : (
          <TableCell align="right" sx={strong('change')}>
            {`${change >= 0 ? '+' : '−'}${duration(Math.abs(change))}`}
          </TableCell>
        )}
        <TableCell sx={strong('why')}>{why(mod)}</TableCell>
      </TableRow>
      <TableRow>
        <TableCell
          colSpan={change === null ? PLAIN_COLUMNS : PLAIN_COLUMNS + 1}
          sx={{ py: 0, borderBottom: open ? undefined : 'none' }}
        >
          <Collapse in={open} unmountOnExit={true}>
            <ModDetails mod={mod} sampled={sampled} />
          </Collapse>
        </TableCell>
      </TableRow>
    </>
  )
}

export function ModTable({
  report,
  previous,
  comparing,
  expanded,
  onToggle,
  onSorted,
}: {
  report: StartupReport
  previous: StartupReport | undefined
  comparing: boolean
  expanded: ReadonlySet<string>
  onToggle: (id: string) => void
  onSorted: () => void
}) {
  const { t } = useLingui()
  const duration = useDuration()
  const { shown, folded } = foldMods(report.mods)
  // Sampled times exist only after a measured launch; they include time inside each mod's patches on game code.
  const sampled = report.sampledOtherMs > 0
  const [sort, setSort] = useStoredState<StoredSort | null>(
    'mortar.startupSort',
    null,
    isStoredSort,
  )
  const active: StoredSort = comparing
    ? { column: 'change', direction: 'desc' }
    : (sort ?? { column: 'total', direction: 'desc' })
  const rows = sortMods(shown, active.column, active.direction, previous)
  const columns: { id: ModSort; label: string; help: string; align?: 'right' }[] = [
    { id: 'name', label: t`Mod`, help: t`Click a mod to show it in Mods` },
    {
      id: 'total',
      label: t`Total`,
      help: t`Entry, event handlers and asset work the SMAPI Bridge timed before the title screen`,
      align: 'right',
    },
    ...(comparing
      ? [
          {
            id: 'change' as const,
            label: t`Change`,
            help: t`Time in this start minus time in the one before`,
            align: 'right' as const,
          },
        ]
      : []),
    {
      id: 'why',
      label: t`Why?`,
      help: t`What costs this mod the most; open the row for every timing`,
    },
  ]
  return (
    <Table size="small" aria-label={t`Startup time by mod`}>
      <TableHead>
        <TableRow>
          {columns.map((column) => (
            <TableCell
              key={column.id}
              align={column.align}
              sortDirection={active.column === column.id ? active.direction : false}
            >
              <Tooltip title={column.help}>
                <TableSortLabel
                  active={active.column === column.id}
                  direction={active.column === column.id ? active.direction : 'desc'}
                  onClick={() => {
                    onSorted()
                    setSort(nextSort(active, column.id))
                  }}
                  sx={{
                    color: 'inherit',
                    '&.Mui-active': { color: 'var(--mortar-ink)' },
                    '& .MuiTableSortLabel-icon': { color: 'inherit !important' },
                  }}
                >
                  {column.label}
                </TableSortLabel>
              </Tooltip>
            </TableCell>
          ))}
        </TableRow>
      </TableHead>
      <TableBody>
        {rows.map((mod) => (
          <ModRow
            key={mod.id}
            mod={mod}
            sampled={sampled}
            sort={active.column}
            open={expanded.has(mod.id)}
            onToggle={() => onToggle(mod.id)}
            change={comparing ? changeOf(mod, previous) : null}
          />
        ))}
        {folded.count > 0 ? (
          <TableRow>
            <TableCell sx={{ color: 'text.secondary', pl: NAME_INDENT }}>
              {plural(folded.count, { one: '# other mod', other: '# other mods' })}
            </TableCell>
            <TableCell align="right">{duration(folded.ms)}</TableCell>
            <TableCell colSpan={comparing ? 2 : 1} />
          </TableRow>
        ) : null}
        <TableRow>
          <TableCell
            sx={{ color: 'text.secondary', pl: NAME_INDENT }}
          >{t`Game and SMAPI`}</TableCell>
          <TableCell align="right">{duration(report.otherMs)}</TableCell>
          <TableCell colSpan={comparing ? 2 : 1} />
        </TableRow>
      </TableBody>
    </Table>
  )
}
