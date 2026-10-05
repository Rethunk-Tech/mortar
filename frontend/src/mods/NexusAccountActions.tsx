import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
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
import { reportUnexpected } from '../toasts/report.ts'
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

  useEffect(() => {
    setStatus(endorsement)
  }, [endorsement])

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
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
      <Button
        size="small"
        disabled={pending || endorsed}
        variant={endorsed ? 'contained' : 'outlined'}
        startIcon={<ThumbsUp size={14} aria-hidden={true} />}
        onClick={() =>
          run(
            async () => {
              setStatus(await Endorse(currentGame(), modId, version))
            },
            { errorTitle: t`Could not endorse` },
          )
        }
      >
        {t`Endorse`}
      </Button>
      <Button
        size="small"
        disabled={pending || abstained}
        variant={abstained ? 'contained' : 'outlined'}
        startIcon={<ThumbsDown size={14} aria-hidden={true} />}
        onClick={() =>
          run(
            async () => {
              setStatus(await Abstain(currentGame(), modId, version))
            },
            { errorTitle: t`Could not abstain` },
          )
        }
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
