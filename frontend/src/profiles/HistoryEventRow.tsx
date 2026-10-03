import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, ListItem, ListItemText } from '@mui/material'
import type { HistoryEvent } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { When } from '../i18n/When.tsx'
import { historyChangeSummary } from './historyCounts.ts'
import type { HistoryDiffView } from './historyDiff.ts'
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
  items: HistoryDiffView['items']
  selected: boolean
  onToggle: () => void
  onUndo: () => void
  onRevertItem: (mod: string) => void
  busy: boolean
}) {
  const { t } = useLingui()
  const changes = historyChangeSummary(ev)
  const marker = ev.kind === 'good'
  return (
    <ListItem
      disableGutters={true}
      secondaryAction={
        <Button size="small" sx={{ whiteSpace: 'nowrap' }} disabled={busy} onClick={onUndo}>
          {t`Undo this change`}
        </Button>
      }
    >
      <Checkbox size="small" checked={selected} onChange={onToggle} disabled={busy} />
      <ListItemText
        primary={marker ? t`Known good` : ev.label}
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
                  sx={{ whiteSpace: 'nowrap' }}
                >
                  {t`Revert`}
                </Button>
              </Box>
            ))}
          </>
        }
      />
    </ListItem>
  )
}
