import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Bell, BellOff, ThumbsDown, ThumbsUp } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  Abstain,
  Endorse,
  Track,
  TrackedMods,
  Untrack,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { currentGame } from '../nav/currentGame.ts'
import { useNexus } from '../settings/nexus.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { errorDetails, errorKind } from '../toasts/errorKind.ts'
import { errorMessage, toastError } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { isAbstained, isEndorsed, isTracked, type TrackedMod } from './nexusAccount.ts'
import { nexusDomain } from './nexusDomain.ts'

export function NexusAccountActions({
  modId,
  version,
  endorsement,
}: {
  modId: number
  version: string
  endorsement: string
}) {
  const { t } = useLingui()
  const signedIn = useNexus((s) => s.signedIn)
  const trackedFail = t`Could not load your tracked Nexus mods`
  const [status, setStatus] = useState(endorsement)
  const { data: mods, setData: setMods } = useLoaded<TrackedMod[] | undefined>(
    signedIn && modId ? () => TrackedMods().then((list) => list ?? []) : null,
    [signedIn, modId],
    undefined,
    (e) => {
      // Offline is the banner's to say; any other failure names what did not load.
      if (errorKind(e) !== 'network') {
        toastError(trackedFail, e)
      }
    },
  )
  const [pending, run] = usePending()
  const [refusal, setRefusal] = useState('')

  useEffect(() => {
    setStatus(endorsement)
    setRefusal('')
  }, [endorsement])

  // Nexus refuses an endorse until the mod was downloaded and played for a while; its own sentence is shown as is.
  const decide = (call: () => Promise<string>) =>
    run(async () => {
      setRefusal('')
      try {
        setStatus(await call())
      } catch (e) {
        setRefusal(errorDetails(e) || errorMessage(e))
      }
    })

  if (!signedIn) {
    return null
  }

  const tracked = isTracked(mods, modId, nexusDomain())
  const endorsed = isEndorsed(status)
  const abstained = isAbstained(status)
  let statusLine = t`You have not endorsed this mod. Nexus asks that you download and play it first.`
  if (endorsed) {
    statusLine = t`You endorsed this mod.`
  } else if (abstained) {
    statusLine = t`You chose not to endorse this mod.`
  }
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: space.gap }}>
      <Typography sx={{ flexBasis: '100%', fontSize: 13, color: 'text.secondary' }}>
        {statusLine}
      </Typography>
      {refusal ? (
        <Typography role="alert" sx={{ flexBasis: '100%', fontSize: 13, color: 'error.main' }}>
          {refusal}
        </Typography>
      ) : null}
      <Button
        size="small"
        disabled={pending || endorsed}
        variant={endorsed ? 'contained' : 'outlined'}
        startIcon={<ThumbsUp size={14} aria-hidden={true} />}
        onClick={() => decide(() => Endorse(currentGame(), modId, version))}
      >
        {t`Endorse`}
      </Button>
      <Button
        size="small"
        disabled={pending || abstained}
        variant={abstained ? 'contained' : 'outlined'}
        startIcon={<ThumbsDown size={14} aria-hidden={true} />}
        onClick={() => decide(() => Abstain(currentGame(), modId, version))}
      >
        {t`Abstain`}
      </Button>
      {tracked ? (
        <Button
          size="small"
          disabled={pending || mods === undefined}
          variant="outlined"
          startIcon={<BellOff size={14} aria-hidden={true} />}
          onClick={() =>
            run(
              async () => {
                await Untrack(currentGame(), modId)
                setMods((cur) => (cur ?? []).filter((m) => m.modId !== modId))
              },
              { errorTitle: t`Could not untrack` },
            )
          }
        >
          {t`Untrack`}
        </Button>
      ) : (
        <Button
          size="small"
          disabled={pending || mods === undefined}
          variant="contained"
          startIcon={<Bell size={14} aria-hidden={true} />}
          onClick={() =>
            run(
              async () => {
                await Track(currentGame(), modId)
                setMods((cur) => [
                  ...(cur ?? []).filter((m) => m.modId !== modId),
                  { modId, domainName: nexusDomain() },
                ])
              },
              { errorTitle: t`Could not track` },
            )
          }
        >
          {t`Track`}
        </Button>
      )}
    </Box>
  )
}
