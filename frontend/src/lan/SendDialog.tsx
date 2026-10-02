import { useLingui } from '@lingui/react/macro'
import {
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Typography,
} from '@mui/material'
import { RefreshCw } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import type { Peer } from '../../bindings/github.com/Rethunk-AI/mortar/internal/lan/models.ts'
import { Peers, Send } from '../../bindings/github.com/Rethunk-AI/mortar/internal/lan/service.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

const refreshInterval = 3000

interface SendDialogProps {
  open: boolean
  game: string
  profileId: string
  onClose: () => void
}

export function SendDialog({ open, game, profileId, onClose }: SendDialogProps) {
  const { t } = useLingui()
  const [peers, setPeers] = useState<Peer[]>([])
  const [refreshing, setRefreshing] = useState(false)
  const [sending, setSending] = useState<string | null>(null)

  const refresh = useCallback(() => {
    setRefreshing(true)
    Peers()
      .then((next) => setPeers(next ?? []))
      .catch((error: unknown) => {
        useToasts.getState().push({
          kind: 'error',
          title: t`Could not find Mortar users`,
          body: errorMessage(error),
        })
      })
      .finally(() => setRefreshing(false))
  }, [t])

  useEffect(() => {
    if (!open) {
      setPeers([])
      return
    }
    refresh()
    const timer = globalThis.setInterval(refresh, refreshInterval)
    return () => globalThis.clearInterval(timer)
  }, [open, refresh])

  const send = (peer: Peer) => {
    setSending(peer.id)
    Send(peer.id, game, profileId)
      .then(() => {
        useToasts.getState().push({ kind: 'success', title: t`Sent to ${peer.name}` })
        onClose()
      })
      .catch((error: unknown) => {
        useToasts.getState().push({
          kind: 'error',
          title: t`Could not send to ${peer.name}`,
          body: errorMessage(error),
        })
      })
      .finally(() => setSending(null))
  }

  return (
    <Dialog open={open} onClose={onClose} fullWidth={true} maxWidth="xs">
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        {t`Send to…`}
        <IconButton aria-label={t`Refresh`} onClick={refresh} disabled={refreshing}>
          {refreshing ? <CircularProgress size={18} /> : <RefreshCw size={18} />}
        </IconButton>
      </DialogTitle>
      <DialogContent dividers={true}>
        {peers.length === 0 && !refreshing ? (
          <Typography color="text.secondary">{t`No Mortar users found nearby.`}</Typography>
        ) : (
          <List disablePadding={true}>
            {peers.map((peer) => (
              <ListItem key={peer.id} disablePadding={true}>
                <ListItemButton disabled={sending !== null} onClick={() => send(peer)}>
                  <ListItemText primary={peer.name} />
                  {sending === peer.id ? <CircularProgress size={18} /> : null}
                </ListItemButton>
              </ListItem>
            ))}
          </List>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
      </DialogActions>
    </Dialog>
  )
}
