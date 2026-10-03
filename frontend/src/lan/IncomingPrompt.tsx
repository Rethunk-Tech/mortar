import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
import {
  CancelTransfer,
  Transfer,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/lan/service.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { openImport } from '../share/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useIncomingShares } from './incoming.ts'

export function IncomingPrompt() {
  const { t } = useLingui()
  const incoming = useIncomingShares((state) => state.items[0])
  const removeFirst = useIncomingShares((state) => state.removeFirst)
  const profiles = useProfiles((state) => state.profiles)
  const incomingGame = incoming?.game ?? ''
  const gameName = useProfiles((state) => {
    if (state.game?.id === incomingGame && state.game.name) {
      return state.game.name
    }
    return incomingGame === 'stardew' ? 'Stardew Valley' : incomingGame
  })
  const progress = useIncomingShares((state) =>
    incoming ? state.progress[incoming.id] : undefined,
  )
  const [choosing, setChoosing] = useState(false)
  const [transferring, setTransferring] = useState(false)
  const [transferError, setTransferError] = useState('')
  const autoAccept = useSettings((s) => s.lanAutoAcceptSameAccount)
  const incomingId = incoming?.id ?? ''
  const sameAccount = incoming?.sameAccount === true

  const accept = async (profileId = '') => {
    if (!incoming) {
      return
    }
    setChoosing(false)
    setTransferError('')
    if (incoming.sameAccount) {
      setTransferring(true)
      try {
        await Transfer(incoming.id)
      } catch (error) {
        setTransferring(false)
        setTransferError(errorMessage(error))
        return
      }
      setTransferring(false)
    }
    openImport(profileId ? { profileId, data: incoming.payload } : { data: incoming.payload })
    removeFirst()
  }

  useEffect(() => {
    if (!(sameAccount && autoAccept && incomingId)) {
      return
    }
    const [item] = useIncomingShares.getState().items
    if (item?.id !== incomingId) {
      return
    }
    let cancelled = false
    setChoosing(false)
    setTransferError('')
    setTransferring(true)
    Transfer(incomingId)
      .then(() => {
        if (cancelled) {
          return
        }
        setTransferring(false)
        openImport({ data: item.payload })
        removeFirst()
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return
        }
        setTransferring(false)
        setTransferError(errorMessage(error))
      })
    return () => {
      cancelled = true
    }
  }, [autoAccept, incomingId, removeFirst, sameAccount])

  if (!incoming) {
    return null
  }
  const decline = () => {
    setChoosing(false)
    removeFirst()
  }
  const compare = (profileId: string) => {
    accept(profileId).catch(reportUnexpected)
  }
  return (
    <>
      <Dialog open={!(choosing || transferring)}>
        <DialogTitle title={incomingGame}>
          {t`${incoming.sender} sent you ${incoming.profileName} (${gameName})`}
        </DialogTitle>
        <DialogContent>
          <Typography color="text.secondary">
            {incoming.sameAccount
              ? t`The mod files can be copied directly from this Mortar.`
              : t`The files will download from each mod's source.`}
          </Typography>
          {transferError ? <Typography color="error">{transferError}</Typography> : null}
        </DialogContent>
        <DialogActions>
          <Button onClick={decline}>{t`Decline`}</Button>
          <DisabledReason title={t`Create a profile first.`} disabled={profiles.length === 0}>
            <Button onClick={() => setChoosing(true)} disabled={profiles.length === 0}>
              {t`Compare with a profile…`}
            </Button>
          </DisabledReason>
          <Button variant="contained" onClick={() => accept()}>
            {t`Import as new profile`}
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={transferring}>
        <DialogTitle>{t`Copying mod files`}</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <CircularProgress size={24} />
            <Typography>
              {progress
                ? t`${progress.current} of ${progress.total} mods · ${formatBytes(progress.bytes)} · ${formatBytes(progress.rate)}/s`
                : t`Preparing transfer…`}
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => {
              CancelTransfer(incoming.id).catch((error: unknown) => {
                useToasts.getState().push({
                  kind: 'error',
                  title: t`Could not cancel transfer`,
                  body: errorMessage(error),
                })
              })
            }}
          >
            {t`Cancel`}
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={choosing} onClose={() => setChoosing(false)} fullWidth={true} maxWidth="xs">
        <DialogTitle>{t`Compare with a profile…`}</DialogTitle>
        <DialogContent dividers={true}>
          <List disablePadding={true}>
            {profiles.map((profile) => (
              <ListItem key={profile.id} disablePadding={true}>
                <ListItemButton onClick={() => compare(profile.id)}>
                  <ListItemText primary={profile.name} />
                </ListItemButton>
              </ListItem>
            ))}
          </List>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setChoosing(false)}>{t`Cancel`}</Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
