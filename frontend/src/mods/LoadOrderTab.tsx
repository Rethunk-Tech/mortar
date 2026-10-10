import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, Link, Skeleton, Tooltip, Typography } from '@mui/material'
import { useVirtualizer } from '@tanstack/react-virtual'
import { Copy, ListOrdered, TriangleAlert } from 'lucide-react'
import { type RefObject, useEffect, useMemo, useRef, useState } from 'react'
import type { Row } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loadorder/models.ts'
import { DependencyNames } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { LoadOrder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { PageActions } from '../game/PageActions.tsx'
import { useTab } from '../game/tab.ts'
import { useGameLoader } from '../games/info.ts'
import { copyText } from '../share/copyText.ts'
import { ControlsRow } from '../shell/ControlsRow.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { idKey, localId } from './dependents.ts'
import { useDetail } from './detail.ts'
import { formatLoadOrderCopy, loadOrderEmptyKind } from './loadOrderText.ts'
import { sameId } from './lookup.ts'
import { requirementNameIn } from './requirementName.ts'
import { useMods } from './store.ts'

const INLINE_REQUIRED = 4
const ROW_ESTIMATE_PX = 72
const INLINE_USERS = 2

function DepChip({
  label,
  title,
  missing = false,
  expanded,
  onClick,
}: {
  label: string
  title?: string
  missing?: boolean
  expanded?: boolean
  onClick?: () => void
}) {
  return (
    <Chip
      size="small"
      variant="outlined"
      label={label}
      title={title}
      color={missing ? 'error' : 'default'}
      icon={missing ? <TriangleAlert size={12} aria-hidden={true} /> : undefined}
      aria-expanded={expanded}
      onClick={onClick}
      sx={{ maxWidth: 240 }}
    />
  )
}

function RowName({ row }: { row: Row }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const listed = mods.find((m) => sameId(m.id, row.id))
  const known = row.name.trim() !== ''
  const label = known ? row.name : localId(row.id) || t`Unknown mod`
  const title = label
  if (!listed) {
    return (
      <Typography noWrap={true} title={title} sx={{ fontWeight: 600, minWidth: 0 }}>
        {label}
      </Typography>
    )
  }
  return (
    <Link
      component="button"
      noWrap={true}
      title={title}
      onClick={() => {
        useTab.getState().setTab('mods')
        useDetail.getState().show(listed)
      }}
      sx={{ fontWeight: 600, minWidth: 0, color: 'text.primary', textAlign: 'left' }}
    >
      {label}
    </Link>
  )
}

function DepChips({
  row,
  names,
  onScroll,
}: {
  row: Row
  names: Map<string, string>
  onScroll: (id: string) => void
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const mods = useMods((s) => s.mods)
  const problems = useMods((s) => s.problems)
  const state = { mods, problems }
  const chip = (prefix: 'req' | 'opt' | 'dep', id: string, missing = false) => {
    const name = (names.get(idKey(id)) ?? '').trim()
    const known = name !== ''
    const label = known ? name : requirementNameIn(state, id)
    const text = {
      req: missing ? t`Needs ${label} (missing)` : t`Required: ${label}`,
      opt: t`Optional: ${label}`,
      dep: t`Used by: ${label}`,
    }[prefix]
    return (
      <DepChip
        key={`${prefix}-${id}`}
        label={text}
        missing={missing}
        {...(known ? {} : { title: id })}
        {...(known && !missing ? { onClick: () => onScroll(id) } : {})}
      />
    )
  }
  const missingIds = new Set((row.missingRequired ?? []).map(idKey))
  const required = row.required ?? []
  const optional = row.optional ?? []
  const users = row.dependents ?? []
  const long = required.length > INLINE_REQUIRED || users.length > INLINE_USERS
  const shownRequired = open ? required : required.slice(0, INLINE_REQUIRED)
  if (required.length + optional.length + users.length === 0) {
    return null
  }
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mt: 0.75 }}>
      {shownRequired.map((id) => chip('req', id, missingIds.has(idKey(id))))}
      {optional.map((id) => chip('opt', id))}
      {open || users.length <= INLINE_USERS ? users.map((id) => chip('dep', id)) : null}
      {long ? (
        <DepChip
          label={
            open
              ? t`Show less`
              : plural(users.length, { one: 'Used by # mod', other: 'Used by # mods' }) +
                (required.length > INLINE_REQUIRED
                  ? t` · ${required.length - INLINE_REQUIRED} more required`
                  : '')
          }
          expanded={open}
          onClick={() => setOpen((v) => !v)}
        />
      ) : null}
    </Box>
  )
}

function OrderList({
  rows,
  names,
  scrollRef,
}: {
  rows: Row[]
  names: Map<string, string>
  scrollRef: RefObject<(id: string) => boolean>
}) {
  const { t } = useLingui()
  const [query, setQuery] = useState('')
  const needle = query.trim().toLowerCase()
  const shown =
    needle === ''
      ? rows
      : rows.filter(
          (row) =>
            row.name.toLowerCase().includes(needle) ||
            localId(row.id).toLowerCase().includes(needle),
        )
  const parentRef = useRef<HTMLDivElement>(null)
  const virtualizer = useVirtualizer({
    count: shown.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => ROW_ESTIMATE_PX,
    overscan: 8,
    getItemKey: (index) => shown[index]?.id ?? index,
  })
  const onScroll = (id: string): boolean => {
    const index = shown.findIndex((row) => sameId(row.id, id))
    if (index < 0) {
      return false
    }
    virtualizer.scrollToIndex(index, { align: 'center' })
    return true
  }
  scrollRef.current = onScroll
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <PageActions>
        <Button
          variant="outlined"
          color="inherit"
          startIcon={<Copy size={16} />}
          onClick={() => {
            copyText(formatLoadOrderCopy(rows), t`Load order copied`)
          }}
        >
          {t`Copy load order`}
        </Button>
      </PageActions>
      <ControlsRow>
        <SearchField value={query} onChange={setQuery} label={t`Filter load order`} grow={true} />
      </ControlsRow>
      <Box
        ref={parentRef}
        sx={{ flex: 1, minHeight: 0, overflow: 'auto', px: space.gutter, pb: space.gutter }}
      >
        <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 1.5 }}>
          {t`The game loader chooses this order. Mortar does not change it.`}
        </Typography>
        <Box sx={{ position: 'relative', height: virtualizer.getTotalSize() }}>
          {virtualizer.getVirtualItems().map((item) => {
            const row = shown[item.index]
            return row ? (
              <Box
                key={item.key}
                data-index={item.index}
                ref={virtualizer.measureElement}
                sx={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  right: 0,
                  transform: `translateY(${item.start}px)`,
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: space.gap,
                  py: space.gap,
                  borderBottom: '1px solid var(--mortar-hairline-faint)',
                  ...(row.cycle
                    ? { outline: '1px solid', outlineColor: 'error.main', outlineOffset: -1 }
                    : {}),
                }}
              >
                <Typography
                  sx={{
                    width: 36,
                    flexShrink: 0,
                    color: 'text.secondary',
                    fontVariantNumeric: 'tabular-nums',
                  }}
                >
                  {row.position}
                </Typography>
                <Box sx={{ minWidth: 0, flex: 1 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minWidth: 0 }}>
                    <RowName row={row} />
                    {row.cycle ? (
                      <Tooltip title={t`These mods require each other.`} describeChild={true}>
                        <Chip
                          size="small"
                          tabIndex={0}
                          label={t`Dependency cycle`}
                          color="error"
                          variant="outlined"
                          sx={{ height: 22 }}
                        />
                      </Tooltip>
                    ) : null}
                    {row.lastRun ? (
                      <Tooltip
                        describeChild={true}
                        title={t`The last launch loaded this here, not where its dependencies put it.`}
                      >
                        <Chip
                          size="small"
                          tabIndex={0}
                          label={t`Last launch`}
                          variant="outlined"
                          sx={{ height: 22 }}
                        />
                      </Tooltip>
                    ) : null}
                  </Box>
                  <DepChips row={row} names={names} onScroll={onScroll} />
                </Box>
              </Box>
            ) : null
          })}
        </Box>
      </Box>
    </Box>
  )
}

const SKELETON_ROWS = 10
const SKELETON_KEYS = Array.from({ length: SKELETON_ROWS }, (_, n) => `row-${n}`)
const SKELETON_NAME = ['40%', '55%', '30%', '48%', '36%']

// Placeholder rows in the list's own shape, so the tab shows where the order will appear while it is read.
function OrderSkeleton({ label }: { label: string }) {
  return (
    <Box
      role="status"
      aria-label={label}
      sx={{ px: space.gutter, pb: space.gutter, display: 'flex', flexDirection: 'column' }}
    >
      <Skeleton variant="rounded" height={34} sx={{ mb: 1.5 }} />
      {SKELETON_KEYS.map((key, i) => (
        <Box
          key={key}
          sx={{
            display: 'flex',
            gap: space.gap,
            py: space.gap,
            borderBottom: '1px solid var(--mortar-hairline-faint)',
          }}
        >
          <Skeleton width={24} />
          <Box sx={{ flex: 1 }}>
            <Skeleton width={SKELETON_NAME[i % SKELETON_NAME.length]} />
            <Box sx={{ display: 'flex', gap: 0.75, mt: 0.75 }}>
              <Skeleton variant="rounded" width={120} height={22} />
              <Skeleton variant="rounded" width={90} height={22} />
            </Box>
          </Box>
        </Box>
      ))}
    </Box>
  )
}

export function LoadOrderTab({ profile, game }: { profile: Profile; game: string }) {
  const { t } = useLingui()
  const loader = useGameLoader(game)
  const mods = useMods((s) => s.mods)
  const enabledKey = mods
    .filter((mod) => mod.enabled)
    .map((mod) => `${mod.id}:${mod.needs?.join(',')}:${mod.optional?.join(',')}`)
    .join('|')
  const scrollRef = useRef<(id: string) => boolean>(() => false)

  const {
    data: loadedRows,
    error,
    reload,
  } = useLoaded<Row[] | null>(
    () => LoadOrder(game, profile.id).then((next) => next ?? []),
    [game, profile.id, enabledKey],
    null,
    reportUnexpected,
  )
  const failed = error !== null
  const rows = failed ? [] : loadedRows

  useEffect(() => {
    if (rows === null) {
      return
    }
    const pending = useTab.getState().takePendingLoadOrder()
    if (!pending) {
      return
    }
    if (!scrollRef.current(pending.id) && pending.fallback !== '') {
      scrollRef.current(pending.fallback)
    }
  }, [rows])

  // Dependencies that are not installed have no row to name them; the dataset names them by their Nexus page.
  const unlisted = useMemo(() => {
    const listed = new Set((rows ?? []).map((row) => idKey(row.id)))
    const ids = (rows ?? []).flatMap((row) => [...(row.required ?? []), ...(row.optional ?? [])])
    return [...new Set(ids.filter((id) => !listed.has(idKey(id))))].sort()
  }, [rows])
  const { data: pageNames } = useLoaded<Record<string, string | undefined>>(
    unlisted.length === 0 ? null : () => DependencyNames(game, unlisted).then((n) => n ?? {}),
    [game, unlisted.join('\0')],
    {},
  )
  const names = useMemo(() => {
    const map = new Map<string, string>()
    for (const [id, name] of Object.entries(pageNames)) {
      map.set(idKey(id), name ?? '')
    }
    for (const row of rows ?? []) {
      map.set(idKey(row.id), row.name)
    }
    return map
  }, [rows, pageNames])

  if (rows === null) {
    return <OrderSkeleton label={t`Reading load order…`} />
  }
  const kind = loadOrderEmptyKind(failed, rows.length)
  if (kind === 'error') {
    return (
      <EmptyState icon={<ListOrdered size={40} />} title={t`Could not read load order`}>
        <Button size="small" onClick={reload}>
          {t`Retry`}
        </Button>
      </EmptyState>
    )
  }
  if (kind === 'empty') {
    return (
      <EmptyState icon={<ListOrdered size={40} />} title={t`No enabled mods`}>
        {t`Enable mods in the Mods tab to see the order ${loader} loads them.`}
      </EmptyState>
    )
  }
  return <OrderList rows={rows} names={names} scrollRef={scrollRef} />
}
