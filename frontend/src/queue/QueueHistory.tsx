import { useLingui } from '@lingui/react/macro'
import { Box, FormControl, MenuItem, Select, Typography } from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { useVirtualizer } from '@tanstack/react-virtual'
import {
  ArrowUpRight,
  CircleCheck,
  CircleX,
  Download,
  Filter,
  RotateCcw,
  Trash2,
  User,
} from 'lucide-react'
import { type ReactNode, useMemo, useRef, useState } from 'react'
import {
  ClearHistory,
  RetryHistory,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { When } from '../i18n/When.tsx'
import { showInProfile } from '../mods/revealMod.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import {
  filterHistory,
  type HistoryEntry,
  type HistoryFilters,
  historyProfiles,
} from './history.ts'

const millisecondsPerSecond = 1000

function RetryHistoryButton({ entry }: { entry: HistoryEntry }) {
  const { t } = useLingui()
  if (entry.outcome !== 'failed' && entry.outcome !== 'skipped' && entry.outcome !== 'cancelled') {
    return null
  }
  return (
    <TipIconButton
      label={t`Download again`}
      onClick={() => RetryHistory(entry).catch(reportUnexpected)}
    >
      <RotateCcw size={16} />
    </TipIconButton>
  )
}

function OutcomeText({ outcome }: { outcome: string }) {
  const { t } = useLingui()
  switch (outcome) {
    case 'done':
      return t`Done`
    case 'failed':
      return t`Failed`
    case 'skipped':
      return t`Skipped`
    case 'cancelled':
      return t`Cancelled`
    default:
      return outcome
  }
}

const ROW_ESTIMATE_PX = 64

// The newest first, rendered only near the viewport: the history keeps up to a thousand downloads.
function HistoryRows({
  rows,
  renderRow,
}: {
  rows: HistoryEntry[]
  renderRow: (e: HistoryEntry) => ReactNode
}) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const newest = useMemo(() => rows.slice().reverse(), [rows])
  const virtualizer = useVirtualizer({
    count: newest.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ROW_ESTIMATE_PX,
    overscan: 8,
  })
  return (
    <Box ref={scrollRef} sx={{ flex: 1, minHeight: 0, overflowY: 'auto' }}>
      <Box sx={{ position: 'relative', height: virtualizer.getTotalSize() }}>
        {virtualizer.getVirtualItems().map((item) => {
          const e = newest[item.index]
          return e ? (
            <Box
              key={item.key}
              data-index={item.index}
              ref={virtualizer.measureElement}
              sx={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                pb: '12px',
                transform: `translateY(${item.start}px)`,
              }}
            >
              {renderRow(e)}
            </Box>
          ) : null
        })}
      </Box>
    </Box>
  )
}

export function HistoryList({
  entries,
  filters,
  onFilters,
  onCleared,
}: {
  entries: HistoryEntry[]
  filters: HistoryFilters
  onFilters: (next: HistoryFilters) => void
  onCleared: () => void
}) {
  const { t } = useLingui()
  const failColor = useTheme().palette.error.light
  const [confirmClear, setConfirmClear] = useState(false)
  const profiles = useProfiles((s) => s.profiles)
  const nameOf = (id: string) => profiles.find((p) => p.id === id)?.name || id
  const rows = filterHistory(entries, filters)
  const ids = historyProfiles(entries)
  return (
    <>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
        <FormControl size="small" sx={{ minWidth: 140 }}>
          <Select
            displayEmpty={true}
            value={filters.outcome}
            onChange={(e) => onFilters({ ...filters, outcome: e.target.value })}
            inputProps={{ 'aria-label': t`Filter by outcome` }}
          >
            <MenuItem value="" sx={{ display: 'flex', gap: 1 }}>
              <Filter size={14} />
              {t`All outcomes`}
            </MenuItem>
            {['done', 'failed', 'skipped', 'cancelled'].map((o) => (
              <MenuItem key={o} value={o} sx={{ display: 'flex', gap: 1 }}>
                {o === 'failed' ? <CircleX size={14} /> : <CircleCheck size={14} />}
                <OutcomeText outcome={o} />
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <FormControl size="small" sx={{ minWidth: 140 }}>
          <Select
            displayEmpty={true}
            value={filters.profileId}
            onChange={(e) => onFilters({ ...filters, profileId: e.target.value })}
            inputProps={{ 'aria-label': t`Filter by profile` }}
          >
            <MenuItem value="" sx={{ display: 'flex', gap: 1 }}>
              <User size={14} />
              {t`All profiles`}
            </MenuItem>
            {ids.map((id) => (
              <MenuItem key={id} value={id} sx={{ display: 'flex', gap: 1 }}>
                <User size={14} />
                {nameOf(id)}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        {entries.length > 0 ? (
          <TipIconButton
            label={t`Clear history`}
            onClick={() => setConfirmClear(true)}
            sx={{ ml: 'auto' }}
          >
            <Trash2 size={16} />
          </TipIconButton>
        ) : null}
        <ConfirmDialog
          open={confirmClear}
          color="error"
          title={t`Clear download history?`}
          body={t`This deletes the list of past downloads. In-progress downloads are not affected. Failed items lose Retry until you download them again.`}
          confirmLabel={t`Clear history`}
          onCancel={() => setConfirmClear(false)}
          onConfirm={() =>
            ClearHistory()
              .then(() => {
                setConfirmClear(false)
                onCleared()
              })
              .catch(reportUnexpected)
          }
        />
      </Box>
      {rows.length === 0 ? (
        <EmptyState compact={true} icon={<Download />} title={t`No downloads in history.`}>
          {t`Completed downloads will appear here.`}
        </EmptyState>
      ) : (
        <HistoryRows
          rows={rows}
          renderRow={(e) => (
            <Box
              key={`${e.started}-${e.finished}-${e.name}-${e.profileId}-${e.outcome}`}
              sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.75 }}
            >
              <Box
                sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 0.25 }}
              >
                <Typography
                  noWrap={true}
                  title={`${e.name}${e.version ? ` ${e.version}` : ''}`}
                  sx={{ fontSize: 14, fontWeight: 600 }}
                >
                  {e.name}
                  {e.version ? ` ${e.version}` : ''}
                </Typography>
                <Typography noWrap={true} sx={{ fontSize: 12, color: 'text.secondary' }}>
                  <OutcomeText outcome={e.outcome} />
                  {` · ${e.source}`}
                  {e.profileId ? ` · ${nameOf(e.profileId)}` : ''}
                  {e.size > 0 ? ` · ${formatBytes(e.size)}` : ''}
                  {e.finished ? (
                    <>
                      {t` · `}
                      <When value={e.finished * millisecondsPerSecond} withTime={true} />
                    </>
                  ) : null}
                </Typography>
                {e.outcome === 'failed' && e.error ? (
                  <Typography noWrap={true} title={e.error} sx={{ fontSize: 12, color: failColor }}>
                    {e.error}
                  </Typography>
                ) : null}
              </Box>
              {e.game && profiles.some((p) => p.id === e.profileId) ? (
                <TipIconButton
                  label={t`Show ${e.name} in its profile`}
                  onClick={() => showInProfile(e.game, e.profileId, e.modId, e.name)}
                >
                  <ArrowUpRight size={16} />
                </TipIconButton>
              ) : null}
              <RetryHistoryButton entry={e} />
            </Box>
          )}
        />
      )}
    </>
  )
}
