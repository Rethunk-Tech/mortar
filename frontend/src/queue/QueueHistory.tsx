import { useLingui } from '@lingui/react/macro'
import { Box, FormControl, IconButton, MenuItem, Select, Tooltip, Typography } from '@mui/material'
import { ArrowUpRight, CircleCheck, CircleX, Download, Filter, Trash2, User } from 'lucide-react'
import { ClearHistory } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { When } from '../i18n/When.tsx'
import { showInProfile } from '../mods/revealMod.ts'
import { useProfiles } from '../profiles/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import {
  filterHistory,
  type HistoryEntry,
  type HistoryFilters,
  historyProfiles,
} from './history.ts'
import { megabytes } from './totals.ts'

const bytesInKb = 1024
const millisecondsPerSecond = 1000

function sizeKb(bytes: number) {
  return Math.floor(bytes / bytesInKb)
}

const menu = {
  transitionDuration: 0,
  PaperProps: { sx: { bgcolor: 'rgb(34,34,42)' } },
} as const

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
            MenuProps={menu}
            sx={{ whiteSpace: 'nowrap' }}
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
            MenuProps={menu}
            sx={{ whiteSpace: 'nowrap' }}
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
          <Tooltip title={t`Clear history`}>
            <IconButton
              aria-label={t`Clear history`}
              onClick={() => ClearHistory().then(onCleared).catch(reportUnexpected)}
              sx={{ ml: 'auto' }}
            >
              <Trash2 size={16} />
            </IconButton>
          </Tooltip>
        ) : null}
      </Box>
      {rows.length === 0 ? (
        <EmptyState compact={true} icon={<Download />} title={t`No downloads in history.`}>
          {t`Completed downloads will appear here.`}
        </EmptyState>
      ) : (
        rows
          .slice()
          .reverse()
          .map((e) => (
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
                  {e.size > 0 ? ` · ${megabytes(sizeKb(e.size))} MB` : ''}
                  {e.finished ? (
                    <>
                      {t` · `}
                      <When value={e.finished * millisecondsPerSecond} withTime={true} />
                    </>
                  ) : null}
                </Typography>
              </Box>
              {e.game && profiles.some((p) => p.id === e.profileId) ? (
                <Tooltip title={t`Show in profile`}>
                  <IconButton
                    size="small"
                    aria-label={t`Show ${e.name} in its profile`}
                    onClick={() => showInProfile(e.game, e.profileId, e.modId, e.name)}
                  >
                    <ArrowUpRight size={16} />
                  </IconButton>
                </Tooltip>
              ) : null}
            </Box>
          ))
      )}
    </>
  )
}
