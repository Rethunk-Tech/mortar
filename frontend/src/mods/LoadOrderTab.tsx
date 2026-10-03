import { useLingui } from '@lingui/react/macro'
import { Box, Chip, Tooltip, Typography } from '@mui/material'
import { ListOrdered } from 'lucide-react'
import { type RefObject, useEffect, useMemo, useRef, useState } from 'react'
import type { Row } from '../../bindings/github.com/Rethunk-AI/mortar/internal/loadorder/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { LoadOrder } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useMods } from './store.ts'

function chipSx(missing: boolean) {
  return {
    height: 22,
    fontSize: 12,
    maxWidth: 220,
    ...(missing ? { color: 'error.main', borderColor: 'error.main' } : {}),
  }
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
  const labelFor = (uniqueId: string) => names.get(uniqueId.toLowerCase()) ?? uniqueId
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mt: 0.75 }}>
      {(row.required ?? []).map((id) => {
        const name = labelFor(id)
        const missing = (row.missingRequired ?? []).some(
          (m) => m.toLowerCase() === id.toLowerCase(),
        )
        const known = names.has(id.toLowerCase())
        return (
          <Chip
            key={`req-${id}`}
            size="small"
            variant="outlined"
            label={t`Required: ${name}`}
            onClick={known && !missing ? () => onScroll(id) : undefined}
            sx={chipSx(missing)}
          />
        )
      })}
      {(row.optional ?? []).map((id) => {
        const name = labelFor(id)
        const known = names.has(id.toLowerCase())
        return (
          <Chip
            key={`opt-${id}`}
            size="small"
            variant="outlined"
            label={t`Optional: ${name}`}
            onClick={known ? () => onScroll(id) : undefined}
            sx={chipSx(false)}
          />
        )
      })}
      {(row.dependents ?? []).map((id) => {
        const name = labelFor(id)
        return (
          <Chip
            key={`dep-${id}`}
            size="small"
            variant="outlined"
            label={t`Used by: ${name}`}
            onClick={() => onScroll(id)}
            sx={chipSx(false)}
          />
        )
      })}
    </Box>
  )
}

function OrderList({
  rows,
  names,
  anchors,
  onScroll,
}: {
  rows: Row[]
  names: Map<string, string>
  anchors: RefObject<Map<string, HTMLElement>>
  onScroll: (id: string) => void
}) {
  const { t } = useLingui()
  return (
    <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto', px: 2, py: 1.5 }}>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 1.5 }}>
        {t`SMAPI chooses this order. Mortar does not change it.`}
      </Typography>
      {rows.map((row) => (
        <Box
          key={row.uniqueId}
          ref={(el: HTMLDivElement | null) => {
            if (el) {
              anchors.current.set(row.uniqueId.toLowerCase(), el)
            } else {
              anchors.current.delete(row.uniqueId.toLowerCase())
            }
          }}
          sx={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: 1.5,
            py: 1,
            borderBottom: '1px solid rgba(255,255,255,0.06)',
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
              <Typography noWrap={true} title={row.name} sx={{ fontWeight: 600, minWidth: 0 }}>
                {row.name}
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
      ))}
    </Box>
  )
}

export function LoadOrderTab({ profile, game }: { profile: Profile; game: string }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const enabledKey = mods
    .filter((mod) => mod.enabled)
    .map((mod) => `${mod.uniqueId}:${mod.needs?.join(',')}:${mod.optional?.join(',')}`)
    .join('|')
  const request = `${game}\0${profile.id}\0${enabledKey}`
  const [rows, setRows] = useState<Row[] | null>(null)
  const anchors = useRef(new Map<string, HTMLElement>())

  useEffect(() => {
    let cancelled = false
    setRows(null)
    const cut = request.indexOf('\0')
    const gameId = request.slice(0, cut)
    const rest = request.slice(cut + 1)
    const cut2 = rest.indexOf('\0')
    LoadOrder(gameId, rest.slice(0, cut2)).then(
      (next) => {
        if (!cancelled) {
          setRows(next ?? [])
        }
      },
      (err) => {
        if (!cancelled) {
          setRows([])
          reportUnexpected(err)
        }
      },
    )
    return () => {
      cancelled = true
    }
  }, [request])

  const names = useMemo(() => {
    const map = new Map<string, string>()
    for (const row of rows ?? []) {
      map.set(row.uniqueId.toLowerCase(), row.name)
    }
    return map
  }, [rows])

  if (rows === null) {
    return <LoadingRow>{t`Reading load order…`}</LoadingRow>
  }
  if (rows.length === 0) {
    return (
      <EmptyState icon={<ListOrdered size={40} />} title={t`No enabled mods`}>
        {t`Switch mods on in the Mods tab to see the order SMAPI loads them.`}
      </EmptyState>
    )
  }
  return (
    <OrderList
      rows={rows}
      names={names}
      anchors={anchors}
      onScroll={(id) => anchors.current.get(id.toLowerCase())?.scrollIntoView({ block: 'center' })}
    />
  )
}
