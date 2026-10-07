import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { History as HistoryIcon } from 'lucide-react'
import { useState } from 'react'
import { listNames } from '../i18n/list.ts'
import { download } from '../queue/actions.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { errorDetails } from '../toasts/errorKind.ts'
import { undoTarget } from '../toasts/history.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { HistoryDiffView } from './HistoryDiffView.tsx'
import { HistoryTimeline } from './HistoryTimeline.tsx'
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
  const [comparing, setComparing] = useState(false)
  // The picks the Compare button was pressed for; changing a pick hides the comparison again.
  const [comparedKey, setComparedKey] = useState('')
  const pair = comparedKey !== '' && comparedKey === h.picked.join(',') ? h.pair : null
  const toggleCompare = () => {
    setComparing(!comparing)
    setComparedKey('')
    h.setPicked([])
  }
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
      maxWidth="md"
      fullWidth={true}
      slotProps={{ paper: { sx: { height: 'min(720px, calc(100% - 64px))' } } }}
    >
      <DialogTitle>{t`History`}</DialogTitle>
      <DialogContent>
        <HistoryToolbar
          busy={!h.game || h.busy !== ''}
          canRestore={h.events.some((ev) => ev.kind === 'good')}
          comparing={comparing}
          picked={h.picked.length}
          onMark={h.markGood}
          onRestore={h.restoreGood}
          onToggleCompare={toggleCompare}
          onCompare={() => setComparedKey(h.picked.join(','))}
        />
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
        {h.events.length === 0 ? (
          <EmptyState icon={<HistoryIcon size={36} />} title={t`No changes yet`}>
            {t`Profile changes will appear here.`}
          </EmptyState>
        ) : (
          <HistoryTimeline
            events={h.events}
            items={h.items}
            comparing={comparing}
            picked={h.picked}
            busy={h.busy !== ''}
            onToggle={(id) =>
              h.setPicked((cur) =>
                cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id].slice(-2),
              )
            }
            onUndo={(id) => h.revertTo(undoTarget(h.events, id)).catch(reportUnexpected)}
            onRevertItem={(id, mod) => h.revertItem(id, mod).catch(reportUnexpected)}
          />
        )}
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
