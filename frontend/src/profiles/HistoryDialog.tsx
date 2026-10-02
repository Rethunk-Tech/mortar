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
import {
  History,
  Revert,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { When } from '../i18n/When.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { historyChangeSummary } from './historyCounts.ts'
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
  const replace = useProfiles((s) => s.replace)
  const [events, setEvents] = useState<HistoryEvent[]>([])
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  useEffect(() => {
    if (!(open && game)) {
      return
    }
    let live = true
    setError('')
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
    setError('')
    try {
      replace(await Revert(game.id, profileId, id))
      setEvents((await History(game.id, profileId)) ?? [])
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy('')
    }
  }
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
                      {t`Revert to here`}
                    </Button>
                  }
                >
                  <ListItemText
                    primary={ev.label}
                    secondary={
                      <>
                        {changes === '' ? null : `${changes} · `}
                        <When value={String(ev.at)} withTime={true} />
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
            <Typography sx={{ fontSize: 13, color: 'error.main' }}>{error}</Typography>
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
