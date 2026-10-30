import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Typography,
} from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
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
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { FormDialog, TextInput } from '../shell/dialog/FormDialog.tsx'
import { sameValue } from '../shell/sameValue.ts'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

function EnterCodeDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  // null until the first scan answers, so the dialog never flashes "none found" before it has looked.
  const { data: found, error: scanError } = useLoaded<Peer[] | null>(
    open ? async () => (await Peers()) ?? [] : null,
    [open],
    null,
    (error) => toastError(t`Could not find Mortar users`, error),
  )
  const peers = scanError === null ? found : []
  const [target, setTarget] = useState('')
  const [manual, setManual] = useState('')
  const [code, setCode] = useState('')
  const [pairing, setPairing] = useState(false)
  useEffect(() => {
    if (!open) {
      setTarget('')
      setManual('')
      setCode('')
    }
  }, [open])
  const address = manual.trim() === '' ? target : manual.trim()
  let issue = ''
  if (address === '') {
    issue = t`Pick the computer showing the code.`
  } else if (code.trim() === '') {
    issue = t`Type the code.`
  }
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
    <FormDialog
      open={open}
      onClose={onClose}
      size="sm"
      title={t`Enter code`}
      submitLabel={t`Pair`}
      issue={issue}
      dirty={code.trim() !== '' || manual.trim() !== ''}
      busy={pairing}
      onSubmit={submit}
    >
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
        <TextInput label={t`Host:port`} value={manual} onChange={setManual} />
      ) : null}
      <TextInput label={t`Code`} value={code} onChange={setCode} />
    </FormDialog>
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
    let live = true
    PairCode()
      .then((next) => live && setCode(next))
      .catch((error: unknown) => {
        if (live) {
          toastError(t`Could not start pairing`, error)
          onCloseRef.current()
        }
      })
    return () => {
      live = false
    }
  }, [open, t])
  return (
    <AppDialog
      open={open}
      onClose={onClose}
      size="sm"
      title={t`Pair a computer`}
      footer={
        <DialogFooter>
          <Button onClick={onClose}>{t`Cancel`}</Button>
        </DialogFooter>
      }
    >
      <Typography color="text.secondary">
        {t`On the other computer, choose Enter code and type this. It works once, for five minutes.`}
      </Typography>
      <Typography variant="h4" sx={{ fontFamily: MONO, textAlign: 'center', pt: space.pad }}>
        {code || '…'}
      </Typography>
    </AppDialog>
  )
}

export function PairedComputers() {
  const { t } = useLingui()
  const [paired, setPaired] = useState<PairedPeer[]>([])
  const [dialog, setDialog] = useState<'show' | 'enter' | null>(null)
  const [unpairing, setUnpairing] = useState<PairedPeer | null>(null)
  const close = useCallback(() => setDialog(null), [])
  const refresh = useCallback(() => {
    Paired()
      .then((next) => setPaired((prev) => (sameValue(prev, next ?? []) ? prev : (next ?? []))))
      .catch(reportUnexpected)
  }, [])
  useEffect(
    () =>
      Events.On('lan:paired', () => {
        refresh()
        setDialog(null)
      }),
    [refresh],
  )
  // Reads the list on mount and again when a dialog closes.
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
        <Box sx={{ display: 'flex', gap: space.gap }}>
          <Button onClick={() => setDialog('show')}>{t`Pair a computer`}</Button>
          <Button onClick={() => setDialog('enter')}>{t`Enter code`}</Button>
        </Box>
      </SettingRow>
      {paired.map((peer) => (
        <SettingRow key={peer.id} label={peer.name}>
          <Button color="error" onClick={() => setUnpairing(peer)}>
            {t`Unpair`}
          </Button>
        </SettingRow>
      ))}
      <ConfirmationDialog
        open={unpairing !== null}
        tone="danger"
        title={t`Unpair ${{ name: unpairing?.name ?? '' }}?`}
        body={t`Profiles sent between you will no longer carry their mod files directly, and you will need a new code to pair again.`}
        confirmLabel={t`Unpair`}
        onCancel={() => setUnpairing(null)}
        onConfirm={() => {
          const id = unpairing?.id
          setUnpairing(null)
          if (id) {
            Unpair(id)
              .then(refresh)
              .catch((error: unknown) => toastError(t`Could not unpair`, error))
          }
        }}
      />
      <ShowCodeDialog open={dialog === 'show'} onClose={close} />
      <EnterCodeDialog open={dialog === 'enter'} onClose={close} />
    </>
  )
}
