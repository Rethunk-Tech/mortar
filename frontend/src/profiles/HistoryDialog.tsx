import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItem,
  ListItemText,
  Typography,
} from '@mui/material'
import { History as HistoryIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { HistoryEvent } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { History } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { When } from '../i18n/When.tsx'
import { download } from '../queue/actions.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { errorDetails, errorMessage, reportUnexpected } from '../toasts/report.ts'
import { historyChangeSummary } from './historyCounts.ts'
import { revertHistoryEvent } from './historyRevert.ts'
import { useProfiles } from './store.ts'

export function HistoryDialog({
  profileId,
  open,
  onClose,
}: {
  profileId: string
  open: boolean
  onClose: () => void
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  const [events, setEvents] = useState<HistoryEvent[]>([])
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const [missingEvent, setMissingEvent] = useState('')
  const [missingNames, setMissingNames] = useState<string[]>([])
  const [missingWants, setMissingWants] = useState<
    Awaited<ReturnType<typeof revertHistoryEvent>>['missingWants']
  >([])
  useEffect(() => {
    if (!(open && game)) {
      return
    }
    let live = true
    setError('')
    setMissingEvent('')
    setMissingNames([])
    setMissingWants([])
    History(game.id, profileId)
      .then((list) => {
        if (live) {
          setEvents(list ?? [])
        }
      })
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [open, game, profileId])
  const revertTo = async (id: string) => {
    if (!game) {
      return
    }
    setBusy(id)
    const result = await revertHistoryEvent(game.id, profileId, id, events)
    setEvents(result.events)
    setError(result.error)
    setMissingEvent(result.missingEvent)
    setMissingNames(result.missingNames)
    setMissingWants(result.missingWants)
    setBusy('')
  }
  const errorText =
    missingNames.length > 0 ? t`Could not restore ${missingNames.join(', ')}` : errorMessage(error)
  return (
    <Dialog
      open={open}
      onClose={onClose}
      transitionDuration={0}
      slotProps={{ paper: { sx: { minWidth: 440, maxWidth: 'calc(100vw - 64px)' } } }}
    >
      <DialogTitle>{t`History`}</DialogTitle>
      <DialogContent>
        {events.length === 0 ? (
          <EmptyState compact={true} icon={<HistoryIcon size={28} />} title={t`No changes yet`}>
            {t`Profile changes will appear here.`}
          </EmptyState>
        ) : (
          <List disablePadding={true}>
            {events.map((ev) => {
              const changes = historyChangeSummary(ev)
              return (
                <ListItem
                  key={ev.id}
                  disableGutters={true}
                  secondaryAction={
                    <Button
                      size="small"
                      sx={{ whiteSpace: 'nowrap' }}
                      disabled={busy !== ''}
                      onClick={() => revertTo(ev.id).catch(reportUnexpected)}
                    >
                      {t`Undo this change`}
                    </Button>
                  }
                >
                  <ListItemText
                    primary={ev.label}
                    secondary={
                      <>
                        {changes === '' ? null : `${changes} · `}
                        <When value={ev.at} withTime={true} />
                      </>
                    }
                  />
                </ListItem>
              )
            })}
          </List>
        )}
        {error !== '' && (
          <Box sx={{ mt: 1 }}>
            <Typography
              sx={{ fontSize: 13, color: 'error.main' }}
              title={missingNames.length > 0 ? undefined : errorDetails(error)}
            >
              {errorText}
            </Typography>
            {missingEvent !== '' && missingWants.length > 0 && (
              <Button
                size="small"
                sx={{ mt: 1, whiteSpace: 'nowrap' }}
                disabled={busy !== ''}
                onClick={() => download(missingWants).catch(reportUnexpected)}
              >
                {t`Download missing mods`}
              </Button>
            )}
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} sx={{ whiteSpace: 'nowrap' }}>
          {t`Close`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
