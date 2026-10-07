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
  ListItemButton,
  ListItemText,
  TextField,
  Typography,
} from '@mui/material'
import { Events } from '@wailsio/runtime'
import { useCallback, useEffect, useRef, useState } from 'react'
import type {
  PairedPeer,
  Peer,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/models.ts'
import {
  Pair,
  PairCode,
  Paired,
  Peers,
  Unpair,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/service.ts'
import { SettingRow } from '../settings/SettingsSection.tsx'
import { reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

function EnterCodeDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  // null until the first scan answers, so the dialog never flashes "none found" before it has looked.
  const [peers, setPeers] = useState<Peer[] | null>(null)
  const [target, setTarget] = useState('')
  const [manual, setManual] = useState('')
  const [code, setCode] = useState('')
  const [pairing, setPairing] = useState(false)
  useEffect(() => {
    if (!open) {
      setTarget('')
      setManual('')
      setCode('')
      setPeers(null)
      return
    }
    Peers()
      .then((next) => setPeers(next ?? []))
      .catch(reportUnexpected)
  }, [open])
  const address = manual.trim() === '' ? target : manual.trim()
  const submit = () => {
    setPairing(true)
    Pair(address, code)
      .then(() => {
        useToasts.getState().push({ kind: 'success', title: t`Computer paired` })
        onClose()
      })
      .catch((error: unknown) => toastError(t`Could not pair`, error))
      .finally(() => setPairing(false))
  }
  return (
    <Dialog open={open} onClose={onClose} fullWidth={true} maxWidth="xs">
      <DialogTitle>{t`Enter code`}</DialogTitle>
      <DialogContent dividers={true}>
        <Typography color="text.secondary">
          {t`Pick the computer showing the code, then type it.`}
        </Typography>
        <List disablePadding={true}>
          {peers?.length === 0 ? (
            <ListItem>
              <ListItemText primary={t`No Mortar computers found nearby.`} />
            </ListItem>
          ) : null}
          {peers?.map((peer) => (
            <ListItem key={peer.id} disablePadding={true}>
              <ListItemButton selected={target === peer.id} onClick={() => setTarget(peer.id)}>
                <ListItemText primary={peer.name} />
              </ListItemButton>
            </ListItem>
          ))}
        </List>
        {peers?.length === 0 ? (
          <TextField
            fullWidth={true}
            size="small"
            label={t`Host:port`}
            value={manual}
            onChange={(event) => setManual(event.target.value)}
            sx={{ mt: 1 }}
          />
        ) : null}
        <TextField
          fullWidth={true}
          size="small"
          label={t`Code`}
          value={code}
          onChange={(event) => setCode(event.target.value)}
          sx={{ mt: 1 }}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
        <Button
          variant="contained"
          disabled={pairing || address === '' || code.trim() === ''}
          onClick={submit}
        >
          {t`Pair`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export function ShowCodeDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const [code, setCode] = useState('')
  // Callers pass a fresh onClose on every render (the peer list re-renders every few seconds); keyed on it, each
  // render would request a new code and replace the one the user is typing.
  const onCloseRef = useRef(onClose)
  onCloseRef.current = onClose
  useEffect(() => {
    if (!open) {
      setCode('')
      return
    }
    PairCode()
      .then(setCode)
      .catch((error: unknown) => {
        toastError(t`Could not start pairing`, error)
        onCloseRef.current()
      })
  }, [open, t])
  return (
    <Dialog open={open} onClose={onClose} fullWidth={true} maxWidth="xs">
      <DialogTitle>{t`Pair a computer`}</DialogTitle>
      <DialogContent>
        <Typography color="text.secondary">
          {t`On the other computer, choose Enter code and type this. It works once, for five minutes.`}
        </Typography>
        <Typography variant="h4" sx={{ fontFamily: 'monospace', textAlign: 'center', py: 2 }}>
          {code || '…'}
        </Typography>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
      </DialogActions>
    </Dialog>
  )
}

export function PairedComputers() {
  const { t } = useLingui()
  const [paired, setPaired] = useState<PairedPeer[]>([])
  const [dialog, setDialog] = useState<'show' | 'enter' | null>(null)
  const close = useCallback(() => setDialog(null), [])
  const refresh = useCallback(() => {
    Paired()
      .then((next) => setPaired(next ?? []))
      .catch(reportUnexpected)
  }, [])
  useEffect(() => {
    refresh()
    return Events.On('lan:paired', () => {
      refresh()
      setDialog(null)
    })
  }, [refresh])
  useEffect(() => {
    if (dialog === null) {
      refresh()
    }
  }, [dialog, refresh])
  return (
    <>
      <SettingRow
        label={t`Paired computers`}
        description={t`Profiles you send between paired computers carry their mod files, whatever the source.`}
      >
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button onClick={() => setDialog('show')}>{t`Pair a computer`}</Button>
          <Button onClick={() => setDialog('enter')}>{t`Enter code`}</Button>
        </Box>
      </SettingRow>
      {paired.map((peer) => (
        <SettingRow key={peer.id} label={peer.name}>
          <Button
            color="error"
            onClick={() => {
              Unpair(peer.id)
                .then(refresh)
                .catch((error: unknown) => toastError(t`Could not unpair`, error))
            }}
          >
            {t`Unpair`}
          </Button>
        </SettingRow>
      ))}
      <ShowCodeDialog open={dialog === 'show'} onClose={close} />
      <EnterCodeDialog open={dialog === 'enter'} onClose={close} />
    </>
  )
}
