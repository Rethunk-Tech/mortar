import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  CircularProgress,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Typography,
} from '@mui/material'
import { useEffect, useRef, useState } from 'react'
import { List as ListGames } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import type {
  Arrival,
  TransferProgress,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/models.ts'
import {
  CancelTransfer,
  Pending,
  Transfer,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/service.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { openImport } from '../share/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
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
    <AppDialog
      open={open}
      onClose={() => undefined}
      closable={false}
      size="sm"
      title={t`Copying mod files`}
      footer={
        <DialogFooter>
          <Button onClick={onCancel}>{t`Cancel`}</Button>
        </DialogFooter>
      }
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.pad }}>
        <CircularProgress size={24} />
        <Typography>
          {progress
            ? [
                plural(
                  { count: progress.total },
                  {
                    one: `${{ current: progress.current }} of # mod`,
                    other: `${{ current: progress.current }} of # mods`,
                  },
                ),
                formatBytes(progress.bytes),
                t`${{ rate: formatBytes(progress.rate) }}/s`,
              ].join(' · ')
            : t`Preparing transfer…`}
        </Typography>
      </Box>
    </AppDialog>
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
    <AppDialog
      open={open}
      onClose={onClose}
      size="sm"
      title={t`Update an existing profile`}
      footer={
        <DialogFooter>
          <Button onClick={onClose}>{t`Cancel`}</Button>
        </DialogFooter>
      }
    >
      <List disablePadding={true}>
        {profiles.map((profile) => (
          <ListItem key={profile.id} disablePadding={true}>
            <ListItemButton onClick={() => onPick(profile.id)}>
              <ListItemText primary={profile.name} />
            </ListItemButton>
          </ListItem>
        ))}
      </List>
    </AppDialog>
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

type Choice = ReturnType<typeof choose>

function OfferButtons({
  incomingGame,
  sender,
  hasProfiles,
  choice,
  onDecline,
  onChoose,
  onAccept,
}: {
  incomingGame: string
  sender: string
  hasProfiles: boolean
  choice: Choice
  onDecline: () => void
  onChoose: () => void
  onAccept: (profileId?: string) => Promise<void>
}) {
  const { t } = useLingui()
  return (
    <>
      <Button onClick={onDecline}>{t`Decline`}</Button>
      <DisabledReason title={t`Create a profile first.`} disabled={!hasProfiles}>
        <Button
          onClick={() => openGameOf(incomingGame).then(onChoose).catch(reportUnexpected)}
          disabled={!hasProfiles}
        >
          {t`Update an existing profile…`}
        </Button>
      </DisabledReason>
      {choice.kind === 'update' ? (
        <>
          <Button onClick={() => onAccept().catch(reportUnexpected)}>
            {t`Import as new profile`}
          </Button>
          <Button
            variant="contained"
            onClick={() => onAccept(choice.profileId).catch(reportUnexpected)}
          >
            {t`Update ${{ name: choice.name }} from ${sender}`}
          </Button>
        </>
      ) : (
        <Button variant="contained" onClick={() => onAccept().catch(reportUnexpected)}>
          {t`Import as new profile`}
        </Button>
      )}
    </>
  )
}

function useAutoAccept({
  paired,
  autoAccept,
  incomingId,
  removeFirst,
  setChoosing,
  setTransferring,
  setTransferError,
}: {
  paired: boolean
  autoAccept: boolean
  incomingId: number | string
  removeFirst: () => void
  setChoosing: (on: boolean) => void
  setTransferring: (on: boolean) => void
  setTransferError: (error: InlineError | null) => void
}) {
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
  }, [autoAccept, incomingId, removeFirst, paired, setChoosing, setTransferring, setTransferError])
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

  // The profile the player chose to update, so a retry after a failed transfer updates it too.
  const chosenId = useRef('')
  const accept = async (profileId = '') => {
    if (!incoming) {
      return
    }
    // Accepted or dismissed elsewhere (`mortar lan accept`) since it arrived: importing it again would make a copy.
    if (!((await Pending()) ?? []).some((a) => a.id === incoming.id)) {
      removeFirst()
      return
    }
    chosenId.current = profileId
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

  useAutoAccept({
    paired,
    autoAccept,
    incomingId,
    removeFirst,
    setChoosing,
    setTransferring,
    setTransferError,
  })

  if (!incoming) {
    return null
  }
  const decline = () => {
    setChoosing(false)
    removeFirst()
  }
  const choice = choose(mapped)
  const { sender } = incoming
  const compare = (profileId: string) => {
    accept(profileId).catch(reportUnexpected)
  }
  return (
    <>
      <AppDialog
        open={!(choosing || transferring)}
        onClose={() => undefined}
        closable={false}
        size="lg"
        title={
          <span title={incomingGame}>
            {t`${{ sender: incoming.sender }} sent you ${{ profile: incoming.profileName }} (${{ game: gameName }})`}
          </span>
        }
        footer={
          <DialogFooter>
            <OfferButtons
              incomingGame={incomingGame}
              sender={sender}
              hasProfiles={profiles.length > 0}
              choice={choice}
              onDecline={decline}
              onChoose={() => setChoosing(true)}
              onAccept={accept}
            />
          </DialogFooter>
        }
      >
        <Typography color="text.secondary">
          {incoming.paired
            ? t`The mod files can be copied directly from this Mortar.`
            : t`The files will download from each mod's source.`}
        </Typography>
        {transferError ? (
          <ErrorRetry
            error={transferError}
            onRetry={() => accept(chosenId.current).catch(reportUnexpected)}
          />
        ) : null}
      </AppDialog>
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
