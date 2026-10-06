import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, ListItem, ListItemText } from '@mui/material'
import type {
  HistoryEvent,
  HistoryItem,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { When } from '../i18n/When.tsx'
import { historyChangeSummary } from './historyCounts.ts'
import { itemModKey } from './historyDiff.ts'
import { historyLabel } from './historyLabel.ts'

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
  // Absent for the oldest event, which nothing older can undo.
  onUndo: (() => void) | undefined
  onRevertItem: (mod: string) => void
  busy: boolean
}) {
  const { t } = useLingui()
  const changes = historyChangeSummary(ev)
  const label = historyLabel(ev)
  const trimmed = ev.kind === 'trimmed'
  return (
    <ListItem
      disableGutters={true}
      secondaryAction={
        trimmed || onUndo === undefined ? null : (
          <Button
            size="small"
            disabled={busy}
            onClick={onUndo}
            aria-label={t`Undo this change: ${label}`}
          >
            {t`Undo`}
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
        primary={label}
        secondary={
          <>
            {changes === '' ? null : `${changes} · `}
            <When value={ev.at} withTime={true} />
            {(items ?? []).map((item) => (
              <Box key={`${item.kind}:${item.mod}:${item.file ?? ''}`} sx={{ mt: 0.5 }}>
                {item.detail}{' '}
                <Button
                  size="small"
                  disabled={busy}
                  onClick={() => onRevertItem(itemModKey(item))}
                  aria-label={t`Restore ${{ label: item.detail }}`}
                >
                  {t`Restore`}
                </Button>
              </Box>
            ))}
          </>
        }
      />
    </ListItem>
  )
}
