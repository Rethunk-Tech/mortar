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
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import { useNexus } from '../settings/nexus.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { isAbstained, isEndorsed, isTracked, type TrackedMod } from './nexusAccount.ts'

const noWrap = { whiteSpace: 'nowrap', textTransform: 'none' } as const

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

  const tracked = isTracked(mods, modId)
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
          run(async () => {
            try {
              setStatus(await Endorse(modId, version))
            } catch (e) {
              reportError(t`Could not endorse`)(e)
            }
          })
        }
        sx={noWrap}
      >
        {t`Endorse`}
      </Button>
      <Button
        size="small"
        disabled={pending || abstained}
        variant={abstained ? 'contained' : 'outlined'}
        startIcon={<ThumbsDown size={14} aria-hidden={true} />}
        onClick={() =>
          run(async () => {
            try {
              setStatus(await Abstain(modId, version))
            } catch (e) {
              reportError(t`Could not abstain`)(e)
            }
          })
        }
        sx={noWrap}
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
            run(async () => {
              try {
                await Untrack(modId)
                setMods((cur) => (cur ?? []).filter((m) => m.modId !== modId))
              } catch (e) {
                reportError(t`Could not untrack`)(e)
              }
            })
          }
          sx={noWrap}
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
            run(async () => {
              try {
                await Track(modId)
                setMods((cur) => [
                  ...(cur ?? []).filter((m) => m.modId !== modId),
                  { modId, domainName: 'stardewvalley' },
                ])
              } catch (e) {
                reportError(t`Could not track`)(e)
              }
            })
          }
          sx={noWrap}
        >
          {t`Track`}
        </Button>
      )}
    </Box>
  )
}
