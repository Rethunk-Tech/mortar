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
import { errorDetails } from '../toasts/errorKind.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
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
  const [status, setStatus] = useState(endorsement)
  const [mods, setMods] = useState<TrackedMod[] | undefined>()
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

  useEffect(() => {
    if (!(signedIn && modId)) {
      setMods(undefined)
      return
    }
    let live = true
    TrackedMods().then(
      (list) => {
        if (live) {
          setMods(list ?? [])
        }
      },
      (e: unknown) => {
        if (live) {
          reportUnexpected(e)
        }
      },
    )
    return () => {
      live = false
    }
  }, [signedIn, modId])

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
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
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
