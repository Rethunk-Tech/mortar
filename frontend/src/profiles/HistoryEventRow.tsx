import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, ListItem, ListItemText } from '@mui/material'
import type {
  HistoryEvent,
  HistoryItem,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { When } from '../i18n/When.tsx'
import { historyChangeSummary } from './historyCounts.ts'
import { itemModKey } from './historyDiff.ts'

export function HistoryEventRow({
  ev,
  items,
  selected,
  onToggle,
  onUndo,
  onRevertItem,
  busy,
}: {
  ev: HistoryEvent
  items: HistoryItem[]
  selected: boolean
  onToggle: () => void
  onUndo: () => void
  onRevertItem: (mod: string) => void
  busy: boolean
}) {
  const { t } = useLingui()
  const changes = historyChangeSummary(ev)
  const marker = ev.kind === 'good'
  const label = marker ? t`Known good` : ev.label
  const trimmed = ev.kind === 'trimmed'
  return (
    <ListItem
      disableGutters={true}
      secondaryAction={
        trimmed ? null : (
          <Button size="small" disabled={busy} onClick={onUndo}>
            {t`Undo ${label}`}
          </Button>
        )
      }
    >
      <Checkbox
        size="small"
        checked={selected}
        onChange={onToggle}
        disabled={busy || trimmed}
        slotProps={{ input: { 'aria-label': t`Compare ${label}` } }}
      />
      <ListItemText
        primary={marker ? t`Known good` : ev.label}
        secondary={
          <>
            {changes === '' ? null : `${changes} · `}
            <When value={ev.at} withTime={true} />
            {(items ?? []).map((item) => (
              <Box key={`${item.kind}:${item.mod}:${item.file ?? ''}`} sx={{ mt: 0.5 }}>
                {item.detail}{' '}
                <Button size="small" disabled={busy} onClick={() => onRevertItem(itemModKey(item))}>
                  {t`Restore ${label}`}
                </Button>
              </Box>
            ))}
          </>
        }
      />
    </ListItem>
  )
}
