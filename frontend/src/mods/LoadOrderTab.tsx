import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, Skeleton, Tooltip, Typography } from '@mui/material'
import { useVirtualizer } from '@tanstack/react-virtual'
import { Clipboard } from '@wailsio/runtime'
import { Copy, ListOrdered } from 'lucide-react'
import { type RefObject, useEffect, useMemo, useRef, useState } from 'react'
import type { Row } from '../../bindings/github.com/Rethunk-AI/mortar/internal/loadorder/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { LoadOrder } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useTab } from '../game/tab.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { formatLoadOrderCopy, loadOrderEmptyKind } from './loadOrderText.ts'
import { sameId } from './lookup.ts'
import { useMods } from './store.ts'

const INLINE_REQUIRED = 4
const ROW_ESTIMATE_PX = 72
const INLINE_USERS = 2

// Thousands of these render at once, so they are plain buttons rather than MUI Chips with tooltips.
function DepChip({
  label,
  title,
  missing = false,
  onClick,
}: {
  label: string
  title?: string
  missing?: boolean
  onClick?: () => void
}) {
  return (
    <Box
      component={onClick ? 'button' : 'span'}
      type={onClick ? 'button' : undefined}
      title={title}
      onClick={onClick}
      sx={{
        display: 'inline-flex',
        alignItems: 'center',
        height: 22,
        maxWidth: 240,
        px: 1,
        border: '1px solid',
        borderColor: missing ? 'error.main' : 'var(--mortar-hairline-22)',
        borderRadius: '11px',
        bgcolor: 'transparent',
        color: missing ? 'error.main' : 'text.primary',
        font: 'inherit',
        fontSize: 12,
        whiteSpace: 'nowrap',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        cursor: onClick ? 'pointer' : 'default',
        '&:hover': onClick ? { bgcolor: 'action.hover' } : {},
      }}
    >
      {label}
    </Box>
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
  const chip = (prefix: 'req' | 'opt' | 'dep', id: string, missing = false) => {
    const name = (names.get(id.toLowerCase()) ?? '').trim()
    const known = name !== ''
    const label = known ? name : t`Unknown mod`
    const text = {
      req: t`Required: ${label}`,
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
  const missingIds = new Set((row.missingRequired ?? []).map((m) => m.toLowerCase()))
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
      {shownRequired.map((id) => chip('req', id, missingIds.has(id.toLowerCase())))}
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
            row.name.toLowerCase().includes(needle) || row.uniqueId.toLowerCase().includes(needle),
        )
  const parentRef = useRef<HTMLDivElement>(null)
  const virtualizer = useVirtualizer({
    count: shown.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => ROW_ESTIMATE_PX,
    overscan: 8,
    getItemKey: (index) => shown[index]?.uniqueId ?? index,
  })
  const onScroll = (id: string): boolean => {
    const index = shown.findIndex((row) => sameId(row.uniqueId, id))
    if (index < 0) {
      return false
    }
    virtualizer.scrollToIndex(index, { align: 'center' })
    return true
  }
  scrollRef.current = onScroll
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 2, pt: 1.5, pb: 0.5 }}>
        <SearchField
          value={query}
          onChange={setQuery}
          label={t`Filter load order`}
          sx={{ flex: 1, minWidth: 0 }}
        />
        <Button
          size="small"
          startIcon={<Copy size={14} />}
          onClick={() => {
            Clipboard.SetText(formatLoadOrderCopy(rows)).then(
              () => useToasts.getState().push({ kind: 'success', title: t`Load order copied` }),
              reportUnexpected,
            )
          }}
        >
          {t`Copy load order`}
        </Button>
      </Box>
      <Box ref={parentRef} sx={{ flex: 1, minHeight: 0, overflow: 'auto', px: 2, py: 1.5 }}>
        <Typography title="SMAPI" sx={{ fontSize: 13, color: 'text.secondary', mb: 1.5 }}>
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
                  gap: 1.5,
                  py: 1,
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
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0 }}>
                    <Typography
                      noWrap={true}
                      title={row.name.trim() === '' ? row.uniqueId : row.name}
                      sx={{ fontWeight: 600, minWidth: 0 }}
                    >
                      {row.name.trim() === '' ? t`Unknown mod` : row.name}
                    </Typography>
                    {row.cycle ? (
                      <Tooltip title={t`These mods require each other.`}>
                        <Chip
                          size="small"
                          label={t`Dependency cycle`}
                          color="error"
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
      sx={{ px: 2, py: 1.5, display: 'flex', flexDirection: 'column' }}
    >
      <Skeleton variant="rounded" height={34} sx={{ mb: 1.5 }} />
      {SKELETON_KEYS.map((key, i) => (
        <Box
          key={key}
          sx={{
            display: 'flex',
            gap: 1.5,
            py: 1,
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
  const mods = useMods((s) => s.mods)
  const [retry, setRetry] = useState(0)
  const enabledKey = mods
    .filter((mod) => mod.enabled)
    .map((mod) => `${mod.uniqueId}:${mod.needs?.join(',')}:${mod.optional?.join(',')}`)
    .join('|')
  const request = `${game}\0${profile.id}\0${enabledKey}\0${retry}`
  const [rows, setRows] = useState<Row[] | null>(null)
  const [failed, setFailed] = useState(false)
  const scrollRef = useRef<(id: string) => boolean>(() => false)

  useEffect(() => {
    let cancelled = false
    setRows(null)
    setFailed(false)
    const cut = request.indexOf('\0')
    const gameId = request.slice(0, cut)
    const rest = request.slice(cut + 1)
    const cut2 = rest.indexOf('\0')
    LoadOrder(gameId, rest.slice(0, cut2)).then(
      (next) => {
        if (!cancelled) {
          setFailed(false)
          setRows(next ?? [])
        }
      },
      (err) => {
        if (!cancelled) {
          setFailed(true)
          setRows([])
          reportUnexpected(err)
        }
      },
    )
    return () => {
      cancelled = true
    }
  }, [request])

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

  const names = useMemo(() => {
    const map = new Map<string, string>()
    for (const row of rows ?? []) {
      map.set(row.uniqueId.toLowerCase(), row.name)
    }
    return map
  }, [rows])

  if (rows === null) {
    return <OrderSkeleton label={t`Reading load order…`} />
  }
  const kind = loadOrderEmptyKind(failed, rows.length)
  if (kind === 'error') {
    return (
      <EmptyState icon={<ListOrdered size={40} />} title={t`Could not read load order`}>
        <Button size="small" onClick={() => setRetry((n) => n + 1)}>
          {t`Retry`}
        </Button>
      </EmptyState>
    )
  }
  if (kind === 'empty') {
    return (
      <EmptyState icon={<ListOrdered size={40} />} title={t`No enabled mods`}>
        {t`Switch mods on in the Mods tab to see the order SMAPI loads them.`}
      </EmptyState>
    )
  }
  return <OrderList rows={rows} names={names} scrollRef={scrollRef} />
}
