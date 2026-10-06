import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  Typography,
} from '@mui/material'
import { History as HistoryIcon } from 'lucide-react'
import { listNames } from '../i18n/list.ts'
import { download } from '../queue/actions.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { errorDetails } from '../toasts/errorKind.ts'
import { undoTarget } from '../toasts/history.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { HistoryDiffView } from './HistoryDiffView.tsx'
import { HistoryEventRow } from './HistoryEventRow.tsx'
import { HistoryToolbar } from './HistoryToolbar.tsx'
import { historyLabel } from './historyLabel.ts'
import { useHistoryPanel } from './useHistoryPanel.ts'

export function HistoryPanel({
  profileId,
  open,
  onClose,
}: {
  profileId: string
  open: boolean
  onClose: () => void
}) {
  const { t } = useLingui()
  const h = useHistoryPanel(profileId, open)
  const { pair } = h
  const labelOf = (id: string) => {
    const ev = h.events.find((e) => e.id === id)
    return ev ? historyLabel(ev) : id
  }
  const errorText =
    h.missingNames.length > 0
      ? t`Could not restore ${listNames(h.missingNames)}`
      : errorMessage(h.error)
  return (
    <Dialog
      open={open}
      onClose={onClose}
      slotProps={{ paper: { sx: { minWidth: 440, maxWidth: 'calc(100vw - 64px)' } } }}
    >
      <DialogTitle>{t`History`}</DialogTitle>
      <DialogContent>
        <HistoryToolbar
          busy={!h.game || h.busy !== ''}
          onMark={h.markGood}
          onRestore={h.restoreGood}
        />
        {h.events.length === 0 ? (
          <EmptyState compact={true} icon={<HistoryIcon size={28} />} title={t`No changes yet`}>
            {t`Profile changes will appear here.`}
          </EmptyState>
        ) : (
          <List disablePadding={true}>
            {h.events.map((ev) => (
              <HistoryEventRow
                key={ev.id}
                ev={ev}
                items={h.items[ev.id] ?? []}
                selected={h.picked.includes(ev.id)}
                busy={h.busy !== ''}
                onToggle={() =>
                  h.setPicked((cur) =>
                    cur.includes(ev.id)
                      ? cur.filter((id) => id !== ev.id)
                      : [...cur, ev.id].slice(-2),
                  )
                }
                onUndo={
                  undoTarget(h.events, ev.id) === ''
                    ? undefined
                    : () => h.revertTo(undoTarget(h.events, ev.id)).catch(reportUnexpected)
                }
                onRevertItem={(mod) => h.revertItem(ev.id, mod).catch(reportUnexpected)}
              />
            ))}
          </List>
        )}
        {pair ? (
          <HistoryDiffView
            diff={pair}
            aLabel={labelOf(pair.a)}
            bLabel={labelOf(pair.b)}
            busy={h.busy !== ''}
            onRestoreA={() => {
              const id = h.pair?.a ?? ''
              if (id !== '') {
                h.revertTo(id).catch(reportUnexpected)
              }
            }}
            onRestoreB={() => {
              const id = h.pair?.b ?? ''
              if (id !== '') {
                h.revertTo(id).catch(reportUnexpected)
              }
            }}
          />
        ) : null}
        {h.error !== '' && (
          <Box sx={{ mt: 1 }}>
            <Typography
              sx={{ fontSize: 13, color: 'error.main' }}
              title={h.missingNames.length > 0 ? undefined : errorDetails(h.error)}
            >
              {errorText}
            </Typography>
            {h.missingEvent !== '' && h.missingWants.length > 0 && (
              <Button
                size="small"
                sx={{ mt: 1 }}
                disabled={h.busy !== ''}
                onClick={() => download(h.missingWants).catch(reportUnexpected)}
              >
                {t`Download missing mods`}
              </Button>
            )}
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Close`}</Button>
      </DialogActions>
    </Dialog>
  )
}
