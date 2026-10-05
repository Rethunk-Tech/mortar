import { useLingui } from '@lingui/react/macro'
import { Box, Button, Card, Snackbar, Typography } from '@mui/material'
import { useEffect, useRef, useState } from 'react'
import { State } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import {
  Endorse,
  EndorsePromptNever,
  EndorsePromptNotNow,
  RecordCleanRun,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useLaunch } from '../launch/store.ts'
import { currentGame } from '../nav/currentGame.ts'
import { useNexus } from '../settings/nexus.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { idKey } from './dependents.ts'
import { useLastRun } from './lastRun.ts'
import { nexusIdOf } from './lookup.ts'
import { useNexusDetails } from './nexusDetails.ts'
import { useMods } from './store.ts'

interface Prompt {
  modId: number
  name: string
  version: string
}

function cleanNexusMods(
  mods: Mod[],
  profile: Profile,
  byId: ReturnType<typeof useLastRun.getState>['byId'],
  details: ReturnType<typeof useNexusDetails.getState>['byId'],
) {
  const seen = new Set<number>()
  const clean: {
    modId: number
    name: string
    version: string
    endorsement: string
  }[] = []
  for (const mod of mods) {
    const modId = nexusIdOf(profile, mod)
    const hit = byId[idKey(mod.uniqueId)]
    if (mod.enabled && modId > 0 && !seen.has(modId) && !hit?.errors) {
      seen.add(modId)
      clean.push({
        modId,
        name: mod.name,
        version: mod.version,
        endorsement: details[modId]?.details?.page.endorsement ?? '',
      })
    }
  }
  return clean
}

export function EndorsePrompt({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const signedIn = useNexus((s) => s.signedIn)
  const loaded = useMods((s) => s.loaded)
  const mods = useMods((s) => s.mods)
  const runId = useLastRun((s) => s.runId)
  const byId = useLastRun((s) => s.byId)
  const details = useNexusDetails((s) => s.byId)
  const launchState = useLaunch((s) => s.status?.state)
  const previousState = useRef(launchState)
  const [completedFrom, setCompletedFrom] = useState<string | null>(null)
  const [prompts, setPrompts] = useState<Prompt[]>([])
  const [busy, setBusy] = useState<number[]>([])

  useEffect(() => {
    if (previousState.current === State.Running && launchState !== State.Running) {
      setCompletedFrom(useLastRun.getState().runId)
    }
    previousState.current = launchState
  }, [launchState])

  useEffect(() => {
    if (!signedIn) {
      setPrompts([])
    }
  }, [signedIn])

  useEffect(() => {
    if (completedFrom === null || !loaded || runId === '' || runId === completedFrom) {
      return
    }
    const clean = cleanNexusMods(mods, profile, byId, details)
    setCompletedFrom(null)
    if (clean.length === 0) {
      return
    }
    RecordCleanRun(runId, clean)
      .then((got) => setPrompts(got ?? []))
      .catch(reportUnexpected)
  }, [byId, completedFrom, details, loaded, mods, profile, runId])

  const run = (modId: number, work: () => Promise<void>) => {
    if (busy.includes(modId)) {
      return
    }
    setBusy((current) => [...current, modId])
    work()
      .then(() => setPrompts((current) => current.filter((prompt) => prompt.modId !== modId)))
      .catch(reportUnexpected)
      .finally(() => setBusy((current) => current.filter((id) => id !== modId)))
  }

  return (
    <Snackbar
      open={signedIn && prompts.length > 0}
      anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
    >
      <Card sx={{ p: 1.5, maxWidth: 460, bgcolor: 'rgb(38,38,46)' }}>
        <Typography sx={{ mb: 1, fontSize: 14, fontWeight: 700 }}>
          {t`Enjoying these mods? Endorse them on Nexus`}
        </Typography>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          {prompts.map((prompt) => (
            <Box key={prompt.modId} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Typography sx={{ flex: 1, minWidth: 0, fontSize: 13, overflowWrap: 'anywhere' }}>
                {t`Enjoying ${prompt.name}? Endorse it on Nexus`}
              </Typography>
              <Button
                size="small"
                variant="contained"
                disabled={busy.includes(prompt.modId)}
                onClick={() =>
                  run(prompt.modId, async () => {
                    await Endorse(currentGame(), prompt.modId, prompt.version)
                    await EndorsePromptNever(prompt.modId)
                  })
                }
              >
                {t`Endorse`}
              </Button>
              <Button
                size="small"
                disabled={busy.includes(prompt.modId)}
                onClick={() => run(prompt.modId, () => EndorsePromptNotNow(prompt.modId))}
              >
                {t`Not now`}
              </Button>
              <Button
                size="small"
                disabled={busy.includes(prompt.modId)}
                onClick={() => run(prompt.modId, () => EndorsePromptNever(prompt.modId))}
              >
                {t`Never for this mod`}
              </Button>
            </Box>
          ))}
        </Box>
      </Card>
    </Snackbar>
  )
}
