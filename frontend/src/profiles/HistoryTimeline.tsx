import { i18n } from '@lingui/core'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useCallback, useRef, useState } from 'react'
import type {
  HistoryDiff,
  HistoryEvent,
  HistoryItem,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { space } from '../theme/density.ts'
import { undoTarget } from '../toasts/history.ts'
import { HistoryEventRow } from './HistoryEventRow.tsx'
import { type HistoryDay, historyDays } from './historyTimeline.ts'

const NO_ITEMS: HistoryItem[] = []
// A long history draws the newest entries and adds older ones a page at a time.
const PAGE = 50

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
  const [shown, setShown] = useState(PAGE)
  // The panel's handlers are new each render; the rows get stable ones that call the latest, so memo holds.
  const latest = useRef({ onToggle, onUndo, onRevertItem })
  latest.current = { onToggle, onUndo, onRevertItem }
  const toggle = useCallback((id: string) => latest.current.onToggle(id), [])
  const undo = useCallback((id: string) => latest.current.onUndo(id), [])
  const revertItem = useCallback(
    (id: string, mod: string) => latest.current.onRevertItem(id, mod),
    [],
  )
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
      {historyDays(events.slice(0, shown)).map((day) => (
        <Box key={day.day} component="section" sx={{ mb: 1.5 }}>
          <Typography
            component="h3"
            sx={{
              fontSize: 'small.fontSize',
              fontWeight: 700,
              color: 'var(--mortar-ink-sec)',
              px: space.gap,
              mb: 0.5,
            }}
          >
            {dayLabel(day)}
          </Typography>
          <Box component="ul" sx={{ m: 0, p: 0 }}>
            {day.events.map((ev) => (
              <HistoryEventRow
                key={ev.id}
                ev={ev}
                items={items[ev.id] ?? NO_ITEMS}
                comparing={comparing}
                selected={picked.includes(ev.id)}
                busy={busy}
                undoable={undoTarget(events, ev.id) !== ''}
                onToggle={toggle}
                onUndo={undo}
                onRevertItem={revertItem}
              />
            ))}
          </Box>
        </Box>
      ))}
      {events.length > shown ? (
        <Button size="small" onClick={() => setShown(shown + PAGE)}>
          {t`Show older changes`}
        </Button>
      ) : null}
    </Box>
  )
}
