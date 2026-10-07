import { plural } from '@lingui/core/macro'
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
import { List as ListGames } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import type {
  Arrival,
  TransferProgress,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/models.ts'
import {
  CancelTransfer,
  Transfer,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/service.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { openImport } from '../share/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { space } from '../theme/density.ts'
import { type InlineError, inlineError, reportUnexpected, toastError } from '../toasts/report.ts'
import { useIncomingShares } from './incoming.ts'
import { choose, lanOriginOf, mappedProfile } from './resume.ts'

// A share imports into its own game and compares against that game's profiles, whatever game is open when it arrives.
async function openGameOf(game: string) {
  if (!isGameId(game)) {
    return
  }
  if (useProfiles.getState().game?.id !== game) {
    await useProfiles.getState().load(game)
  }
  useNav.getState().openGame(game)
}

// The sender's profile, for remembering where this share lands.
function lanOf(arrival: Arrival) {
  const lan = lanOriginOf(arrival)
  return lan ? { lan } : {}
}

function CopyingDialog({
  open,
  progress,
  onCancel,
}: {
  open: boolean
  progress: TransferProgress | undefined
  onCancel: () => void
}) {
  const { t } = useLingui()
  return (
    <Dialog open={open}>
      <DialogTitle>{t`Copying mod files`}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: space.pad }}>
          <CircularProgress size={24} />
          <Typography>
            {progress
              ? plural(progress.total, {
                  one: `${progress.current} of # mod · ${formatBytes(progress.bytes)} · ${formatBytes(progress.rate)}/s`,
                  other: `${progress.current} of # mods · ${formatBytes(progress.bytes)} · ${formatBytes(progress.rate)}/s`,
                })
              : t`Preparing transfer…`}
          </Typography>
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onCancel}>{t`Cancel`}</Button>
      </DialogActions>
    </Dialog>
  )
}

function ChooseDialog({
  open,
  profiles,
  onPick,
  onClose,
}: {
  open: boolean
  profiles: { id: string; name: string }[]
  onPick: (profileId: string) => void
  onClose: () => void
}) {
  const { t } = useLingui()
  return (
    <Dialog open={open} onClose={onClose} fullWidth={true} maxWidth="xs">
      <DialogTitle>{t`Update an existing profile…`}</DialogTitle>
      <DialogContent dividers={true}>
        <List disablePadding={true}>
          {profiles.map((profile) => (
            <ListItem key={profile.id} disablePadding={true}>
              <ListItemButton onClick={() => onPick(profile.id)}>
                <ListItemText primary={profile.name} />
              </ListItemButton>
            </ListItem>
          ))}
        </List>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
      </DialogActions>
    </Dialog>
  )
}

// The profile an earlier share of the same sender profile went into. The profile list is the open game's, so a
// share for another game cannot be matched until that game is open.
function useResumedProfile(arrival: Arrival | undefined) {
  const profiles = useProfiles((state) => state.profiles)
  const loadedGame = useProfiles((state) => state.game?.id)
  return mappedProfile(
    arrival && loadedGame === arrival.game ? lanOriginOf(arrival) : undefined,
    profiles,
  )
}

function useGameName(game: string): string {
  const [listedName, setListedName] = useState('')
  useEffect(() => {
    ListGames()
      .then((games) => setListedName((games ?? []).find((g) => g.id === game)?.name ?? ''))
      .catch(reportUnexpected)
  }, [game])
  return listedName || game
}

export function IncomingPrompt() {
  const { t } = useLingui()
  const incoming = useIncomingShares((state) => state.items[0])
  const removeFirst = useIncomingShares((state) => state.removeFirst)
  const profiles = useProfiles((state) => state.profiles)
  const incomingGame = incoming?.game ?? ''
  const gameName = useGameName(incomingGame)
  const progress = useIncomingShares((state) =>
    incoming ? state.progress[incoming.id] : undefined,
  )
  const [choosing, setChoosing] = useState(false)
  const [transferring, setTransferring] = useState(false)
  const [transferError, setTransferError] = useState<InlineError | null>(null)
  const mapped = useResumedProfile(incoming)
  const autoAccept = useSettings((s) => s.lanAutoAcceptPaired)
  const incomingId = incoming?.id ?? ''
  const paired = incoming?.paired === true

  const accept = async (profileId = '') => {
    if (!incoming) {
      return
    }
    setChoosing(false)
    setTransferError(null)
    if (incoming.paired) {
      setTransferring(true)
      try {
        await Transfer(incoming.id)
      } catch (error) {
        setTransferring(false)
        setTransferError(inlineError(error))
        return
      }
      setTransferring(false)
    }
    await openGameOf(incoming.game)
    openImport({ ...(profileId ? { profileId } : {}), data: incoming.payload, ...lanOf(incoming) })
    removeFirst()
  }

  useEffect(() => {
    if (!(paired && autoAccept && incomingId)) {
      return
    }
    const [item] = useIncomingShares.getState().items
    if (item?.id !== incomingId) {
      return
    }
    let cancelled = false
    setChoosing(false)
    setTransferError(null)
    setTransferring(true)
    Transfer(incomingId)
      .then(() => openGameOf(item.game))
      .then(() => {
        if (cancelled) {
          return
        }
        setTransferring(false)
        const resumed = mappedProfile(lanOriginOf(item), useProfiles.getState().profiles)
        openImport({
          ...(resumed ? { profileId: resumed.id } : {}),
          data: item.payload,
          ...lanOf(item),
        })
        removeFirst()
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return
        }
        setTransferring(false)
        setTransferError(inlineError(error))
      })
    return () => {
      cancelled = true
    }
  }, [autoAccept, incomingId, removeFirst, paired])

  if (!incoming) {
    return null
  }
  const decline = () => {
    setChoosing(false)
    removeFirst()
  }
  const choice = choose(mapped)
  const localName = choice.kind === 'update' ? choice.name : ''
  const { sender } = incoming
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
            {incoming.paired
              ? t`The mod files can be copied directly from this Mortar.`
              : t`The files will download from each mod's source.`}
          </Typography>
          {transferError ? (
            <ErrorRetry error={transferError} onRetry={() => accept().catch(reportUnexpected)} />
          ) : null}
        </DialogContent>
        <DialogActions>
          <Button onClick={decline}>{t`Decline`}</Button>
          <DisabledReason title={t`Create a profile first.`} disabled={profiles.length === 0}>
            <Button
              onClick={() =>
                openGameOf(incomingGame)
                  .then(() => setChoosing(true))
                  .catch(reportUnexpected)
              }
              disabled={profiles.length === 0}
            >
              {t`Update an existing profile…`}
            </Button>
          </DisabledReason>
          {choice.kind === 'update' ? (
            <>
              <Button onClick={() => accept().catch(reportUnexpected)}>
                {t`Import as new profile`}
              </Button>
              <Button variant="contained" onClick={() => compare(choice.profileId)}>
                {t`Update ${localName} from ${sender}`}
              </Button>
            </>
          ) : (
            <Button variant="contained" onClick={() => accept()}>
              {t`Import as new profile`}
            </Button>
          )}
        </DialogActions>
      </Dialog>
      <CopyingDialog
        open={transferring}
        progress={progress}
        onCancel={() => {
          CancelTransfer(incoming.id).catch((error: unknown) => {
            toastError(t`Could not cancel transfer`, error)
          })
        }}
      />
      <ChooseDialog
        open={choosing}
        profiles={profiles}
        onPick={compare}
        onClose={() => setChoosing(false)}
      />
    </>
  )
}
