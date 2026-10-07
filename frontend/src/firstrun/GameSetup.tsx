import { useLingui } from '@lingui/react/macro'
import { Box, Button, Step, StepLabel, Stepper, Typography } from '@mui/material'
import { ArrowLeft } from 'lucide-react'

import { useCallback, useEffect, useRef, useState } from 'react'
import type {
  GameInfo,
  StoreApp,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import {
  Launchers,
  List,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { useGameName } from '../games/info.ts'
import { useLoader } from '../loader/store.ts'
import { type GameId, useNav } from '../nav/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { LoadErrorRow, LoadingRow } from '../shell/LoadingRow.tsx'
import { type InlineError, inlineError } from '../toasts/report.ts'
import { FindStep } from './FindStep.tsx'
import { LoaderStep } from './LoaderStep.tsx'
import { loaderChip } from './logic.ts'
import { NexusStep } from './NexusStep.tsx'
import { ProfileStep } from './ProfileStep.tsx'

const FIND = 1
const NEXUS = 2
const LOADER = 3
const PROFILE = 4
type SetupStep = typeof FIND | typeof NEXUS | typeof LOADER | typeof PROFILE

// Leaves the setup for the game that was open before it, or for the game list when there was none.
function BackButton() {
  const { t } = useLingui()
  const from = useNav((s) => (s.route.name === 'game-setup' ? s.route.from : undefined))
  const name = useGameName(from)
  return (
    <Button
      color="inherit"
      startIcon={<ArrowLeft size={16} />}
      onClick={() => {
        const nav = useNav.getState()
        if (from) {
          nav.openGame(from)
        } else {
          nav.openGameSelect()
        }
      }}
    >
      {from ? t`Back to ${name}` : t`All games`}
    </Button>
  )
}

function SetupStepper({
  step,
  loader,
  nexus,
  loaderInstalled,
}: {
  step: SetupStep
  loader: string | null
  nexus: boolean
  loaderInstalled: boolean
}) {
  const { t } = useLingui()
  const loaderState = loaderChip(step, LOADER, loaderInstalled)
  const stateOf = (n: SetupStep) => {
    if (n < step) {
      return 'done'
    }
    return n === step ? 'current' : 'todo'
  }
  const steps = [
    { key: 'find', label: t`Game folder`, state: stateOf(FIND) },
    ...(nexus ? [{ key: 'nexus', label: t`Nexus Mods`, state: stateOf(NEXUS) }] : []),
    ...(loader === null
      ? []
      : [
          {
            key: 'loader',
            label: loaderState === 'done' ? t`${{ name: loader }} installed` : t`Install ${loader}`,
            state: loaderState,
          },
        ]),
    { key: 'profile', label: t`First profile`, state: stateOf(PROFILE) },
  ]
  return (
    <Stepper
      activeStep={steps.findIndex((x) => x.state === 'current')}
      sx={{ width: 'min(640px, 90%)' }}
    >
      {steps.map((x) => (
        <Step key={x.key} completed={x.state === 'done'}>
          <StepLabel>{x.label}</StepLabel>
        </Step>
      ))}
    </Stepper>
  )
}

// Sets up one game the first time it is opened: its folder, its loader, then a first profile.
export function GameSetup({ game: id }: { game: GameId }) {
  const { t } = useLingui()
  const [step, setStep] = useState<SetupStep>(FIND)
  const [game, setGame] = useState<GameInfo | null>(null)
  const [launchers, setLaunchers] = useState<StoreApp[]>([])
  const [loadError, setLoadError] = useState<InlineError | null>(null)
  const [ready, setReady] = useState(false)
  const loaded = useRef(false)
  // Only the first read shows Loading; a refresh on window focus keeps the step and its row errors.
  const refresh = useCallback(() => {
    if (!loaded.current) {
      setLoadError(null)
      setReady(false)
    }
    Promise.all([List(), Launchers()])
      .then(([games, ls]) => {
        loaded.current = true
        setGame((games ?? []).find((g) => g.id === id) ?? null)
        setLaunchers(ls ?? [])
      })
      .catch((e: unknown) => {
        if (!loaded.current) {
          setLoadError(inlineError(e))
        }
      })
      .finally(() => setReady(true))
  }, [id])
  useEffect(refresh, [refresh])
  const goToProfile = useCallback(() => setStep(PROFILE), [])
  const hasLoader = Boolean(game?.loaderId)
  const signedIn = useNexus((s) => s.signedIn)
  const loaderInstalled = useLoader(
    (s) => s.game === id && s.status?.installed === true && !s.status.broken,
  )
  const afterNexus = hasLoader ? LOADER : PROFILE
  // Nexus's Mod Manager Download button only matters to a game whose mods come from Nexus first; a Thunderstore
  // game gets its mods in Mortar's own Browse, and Settings has the sign-in for later.
  const nexusFirst = (game?.sources ?? [])[0] === 'nexus'
  const afterFind = signedIn || !nexusFirst ? afterNexus : NEXUS
  const firstStep = useRef(step)
  // The step's content remounts on each change, dropping focus to the page; a remounted wrapper takes it back.
  const focusStep = (el: HTMLDivElement | null) => {
    if (step !== firstStep.current) {
      el?.focus()
    }
  }

  if (!ready) {
    return <LoadingRow>{t`Loading…`}</LoadingRow>
  }
  if (loadError || !game) {
    const alert = loadError ?? inlineError(null)
    return <LoadErrorRow error={alert} onRetry={refresh} />
  }
  return (
    <Box
      sx={{
        height: '100%',
        overflowY: 'auto',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: '20px',
        pt: '34px',
        pb: 3,
      }}
    >
      <Box sx={{ alignSelf: 'flex-start', ml: 2, mt: -2 }}>
        <BackButton />
      </Box>
      <Typography component="h1" sx={{ fontSize: 34, fontWeight: 700 }}>
        {t`Set up ${game.name}`}
      </Typography>
      <SetupStepper
        step={step}
        loader={hasLoader ? game.loader : null}
        nexus={nexusFirst}
        loaderInstalled={loaderInstalled}
      />
      <Box
        key={step}
        ref={focusStep}
        tabIndex={-1}
        sx={{
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: '20px',
          outline: 'none',
        }}
      >
        {step === FIND ? (
          <FindStep
            game={game}
            launchers={launchers}
            refresh={refresh}
            onContinue={() => setStep(afterFind)}
          />
        ) : null}
        {step === NEXUS ? <NexusStep onDone={() => setStep(afterNexus)} /> : null}
        {step === LOADER && hasLoader ? (
          <LoaderStep
            game={id}
            loader={game.loader}
            gameDir={game.installDir}
            onDone={goToProfile}
          />
        ) : null}
        {step === PROFILE ? (
          <ProfileStep game={id} site={sourceLabel(game.sources?.[0] ?? 'nexus')} />
        ) : null}
      </Box>
    </Box>
  )
}
