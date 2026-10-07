import { useLingui } from '@lingui/react/macro'
import {
  Box,
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
  TextField,
} from '@mui/material'
import { Events } from '@wailsio/runtime'
import { Inbox, RefreshCw } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import type { Peer } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/models.ts'
import { Peers, Send } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/service.ts'
import { useSettings } from '../settings/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { errorKind } from '../toasts/errorKind.ts'
import { toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { ShowCodeDialog } from './PairedComputers.tsx'
import { peerSecondary } from './peerSecondary.ts'

const refreshInterval = 3000

function sendPeerView(
  looked: boolean,
  refreshing: boolean,
  peerCount: number,
): 'empty' | 'looking' | 'list' {
  if (peerCount === 0 && looked && !refreshing) {
    return 'empty'
  }
  if (peerCount === 0 && !looked) {
    return 'looking'
  }
  return 'list'
}

interface SendDialogProps {
  open: boolean
  game: string
  profileId: string
  onClose: () => void
}

interface SendTarget {
  id: string
  name: string
}

interface PeerRowProps {
  peer: Peer
  sharedName: boolean
  disabled: boolean
  sending: boolean
  onSend: () => void
  onPair: () => void
}

function PeerRow({ peer, sharedName, disabled, sending, onSend, onPair }: PeerRowProps) {
  const { t } = useLingui()
  return (
    <ListItem disablePadding={true}>
      <ListItemButton disabled={disabled} onClick={onSend}>
        <ListItemText
          primary={peer.name}
          secondary={peerSecondary(
            peer,
            sharedName,
            t`Paired · sends mod files`,
            t`Not paired · they download each mod`,
          )}
        />
        {peer.paired ? null : (
          <Button
            size="small"
            variant="text"
            onClick={(event) => {
              event.stopPropagation()
              onPair()
            }}
          >
            {t`Pair`}
          </Button>
        )}
        {sending ? <CircularProgress size={18} /> : null}
      </ListItemButton>
    </ListItem>
  )
}

export function SendDialog({ open, game, profileId, onClose }: SendDialogProps) {
  const { t } = useLingui()
  const [peers, setPeers] = useState<Peer[]>([])
  const [refreshing, setRefreshing] = useState(false)
  const [looked, setLooked] = useState(false)
  const [sending, setSending] = useState<string | null>(null)
  const [manual, setManual] = useState('')
  const [addressPicker, setAddressPicker] = useState(false)
  const [pairing, setPairing] = useState(false)
  const addresses = useSettings((state) => state.lanAddresses ?? [])

  const refresh = useCallback(() => {
    setRefreshing(true)
    Peers()
      .then((next) => setPeers(next ?? []))
      .catch((error: unknown) => {
        toastError(t`Could not find Mortar users`, error)
      })
      .finally(() => {
        setRefreshing(false)
        setLooked(true)
      })
  }, [t])

  useEffect(() => {
    if (!open) {
      setPeers([])
      setManual('')
      setAddressPicker(false)
      setLooked(false)
      return
    }
    refresh()
    const timer = globalThis.setInterval(refresh, refreshInterval)
    return () => globalThis.clearInterval(timer)
  }, [open, refresh])

  useEffect(
    () =>
      Events.On('lan:paired', () => {
        setPairing(false)
        refresh()
      }),
    [refresh],
  )

  const send = (target: SendTarget) => {
    setSending(target.id)
    Send(target.id, game, profileId)
      .then(() => {
        useToasts.getState().push({ kind: 'success', title: t`Sent to ${target.name}` })
        onClose()
      })
      .catch((error: unknown) => {
        // Busy here is the other Mortar turning away a second share that came within seconds of the last.
        if (errorKind(error) === 'busy') {
          useToasts.getState().push({
            kind: 'error',
            title: t`${target.name} is still taking your last share`,
            body: t`Try again in a few seconds.`,
            action: { label: t`Retry`, run: () => send(target) },
          })
          return
        }
        toastError(t`Could not send to ${target.name}`, error)
      })
      .finally(() => setSending(null))
  }

  const sharedNames = new Set(
    peers.map((peer) => peer.name).filter((name, index, all) => all.indexOf(name) !== index),
  )

  return (
    <Dialog open={open} onClose={onClose} fullWidth={true} maxWidth="xs">
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        {t`Send to…`}
        <IconButton aria-label={t`Refresh`} onClick={refresh} disabled={refreshing}>
          {refreshing ? <CircularProgress size={18} /> : <RefreshCw size={18} />}
        </IconButton>
      </DialogTitle>
      <DialogContent dividers={true}>
        {sendPeerView(looked, refreshing, peers.length) === 'empty' ? (
          <EmptyState
            compact={true}
            icon={<Inbox size={28} />}
            title={t`No Mortar users found nearby.`}
          >{t`Ask another Mortar user to open sharing nearby.`}</EmptyState>
        ) : (
          <List disablePadding={true}>
            {sendPeerView(looked, refreshing, peers.length) === 'looking' ? (
              <ListItem>
                <ListItemText primary={t`Looking for Mortar users…`} />
              </ListItem>
            ) : null}
            {peers.map((peer) => (
              <PeerRow
                key={peer.id}
                peer={peer}
                sharedName={sharedNames.has(peer.name)}
                disabled={sending !== null}
                sending={sending === peer.id}
                onSend={() => send(peer)}
                onPair={() => setPairing(true)}
              />
            ))}
            <ListItem disablePadding={true}>
              <ListItemButton
                disabled={sending !== null}
                onClick={() => setAddressPicker((current) => !current)}
              >
                <ListItemText primary={t`Send to an address…`} />
              </ListItemButton>
            </ListItem>
            {addresses.map((address) => (
              <ListItem key={address} disablePadding={true}>
                <ListItemButton
                  disabled={sending !== null}
                  onClick={() => send({ id: address, name: address })}
                >
                  <ListItemText primary={address} secondary={t`Recent address`} />
                </ListItemButton>
              </ListItem>
            ))}
          </List>
        )}
        {addressPicker ? (
          <Box sx={{ display: 'flex', gap: 1, mt: 1 }}>
            <TextField
              autoFocus={true}
              fullWidth={true}
              size="small"
              label={t`Host:port`}
              value={manual}
              onChange={(event) => setManual(event.target.value)}
            />
            <Button
              variant="contained"
              disabled={sending !== null || manual.trim() === ''}
              onClick={() => send({ id: manual.trim(), name: manual.trim() })}
            >
              {t`Send`}
            </Button>
          </Box>
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
      </DialogActions>
      <ShowCodeDialog open={pairing} onClose={() => setPairing(false)} />
    </Dialog>
  )
}
