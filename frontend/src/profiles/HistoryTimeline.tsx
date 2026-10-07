import { i18n } from '@lingui/core'
import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import type {
  HistoryDiff,
  HistoryEvent,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { undoTarget } from '../toasts/history.ts'
import { HistoryEventRow } from './HistoryEventRow.tsx'
import { type HistoryDay, historyDays } from './historyTimeline.ts'

export function HistoryTimeline({
  events,
  items,
  comparing,
  picked,
  busy,
  onToggle,
  onUndo,
  onRevertItem,
}: {
  events: HistoryEvent[]
  items: Record<string, HistoryDiff['items']>
  comparing: boolean
  picked: string[]
  busy: boolean
  onToggle: (id: string) => void
  onUndo: (id: string) => void
  onRevertItem: (id: string, mod: string) => void
}) {
  const { t } = useLingui()
  const dayLabel = (day: HistoryDay<unknown>) => {
    if (day.label === 'today') {
      return t`Today`
    }
    if (day.label === 'yesterday') {
      return t`Yesterday`
    }
    return new Intl.DateTimeFormat(i18n.locale, {
      weekday: 'long',
      month: 'long',
      day: 'numeric',
    }).format(new Date(day.at))
  }
  return (
    <Box>
      {historyDays(events).map((day) => (
        <Box key={day.day} component="section" sx={{ mb: 1.5 }}>
          <Typography
            component="h3"
            sx={{ fontSize: 12, fontWeight: 700, color: 'var(--mortar-ink-sec)', px: 1, mb: 0.5 }}
          >
            {dayLabel(day)}
          </Typography>
          <Box component="ul" sx={{ m: 0, p: 0 }}>
            {day.events.map((ev) => (
              <HistoryEventRow
                key={ev.id}
                ev={ev}
                items={items[ev.id] ?? []}
                comparing={comparing}
                selected={picked.includes(ev.id)}
                busy={busy}
                onToggle={() => onToggle(ev.id)}
                onUndo={undoTarget(events, ev.id) === '' ? undefined : () => onUndo(ev.id)}
                onRevertItem={(mod) => onRevertItem(ev.id, mod)}
              />
            ))}
          </Box>
        </Box>
      ))}
    </Box>
  )
}
