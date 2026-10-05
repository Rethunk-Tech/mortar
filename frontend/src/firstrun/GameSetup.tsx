import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { alpha } from '@mui/material/styles'
import { Check } from 'lucide-react'

const DONE_FILL = 0.18

import { useCallback, useEffect, useRef, useState } from 'react'
import type {
  GameInfo,
  StoreApp,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import {
  Launchers,
  List,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import type { GameId } from '../nav/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { LoadErrorRow, LoadingRow } from '../shell/LoadingRow.tsx'
import { type InlineError, inlineError } from '../toasts/report.ts'
import { FindStep } from './FindStep.tsx'
import { LoaderStep } from './LoaderStep.tsx'
import { NexusStep } from './NexusStep.tsx'
import { ProfileStep } from './ProfileStep.tsx'

const FIND = 1
const NEXUS = 2
const LOADER = 3
const PROFILE = 4
type Step = typeof FIND | typeof NEXUS | typeof LOADER | typeof PROFILE

function StepChip({
  n,
  label,
  state,
}: {
  n: number
  label: string
  state: 'done' | 'current' | 'todo'
}) {
  const { t } = useLingui()
  return (
    <Box
      component="li"
      aria-current={state === 'current' ? 'step' : undefined}
      sx={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 0.75,
        px: '14px',
        py: '6px',
        borderRadius: '16px',
        fontSize: 14,
        fontWeight: state === 'todo' ? 'normal' : 'bold',
        whiteSpace: 'nowrap',
        bgcolor: (theme) =>
          ({
            done: alpha(theme.palette.success.main, DONE_FILL),
            current: theme.palette.primary.main,
            todo: 'var(--mortar-hairline)',
          })[state],
        color: (theme) =>
          ({
            done: theme.palette.success.light,
            current: theme.palette.primary.contrastText,
            todo: theme.palette.text.secondary,
          })[state],
      }}
    >
      {state === 'done' ? (
        <Check size={14} strokeWidth={2.5} role="img" aria-label={t`Done`} />
      ) : (
        String(n)
      )}{' '}
      {label}
    </Box>
  )
}

// Sets up one game the first time it is opened: its folder, its loader, then a first profile.
export function GameSetup({ game: id }: { game: GameId }) {
  const { t } = useLingui()
  const [step, setStep] = useState<Step>(FIND)
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
  const afterNexus = hasLoader ? LOADER : PROFILE
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
    const alert = loadError ?? inlineError(null, t`Something went wrong.`)
    return <LoadErrorRow error={alert} onRetry={refresh} />
  }
  const stateOf = (n: Step) => {
    if (n < step) {
      return 'done'
    }
    return n === step ? 'current' : 'todo'
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
      <Typography component="h1" sx={{ fontSize: 34, fontWeight: 700 }}>
        {t`Set up ${game.name}`}
      </Typography>
      <Box component="ol" sx={{ display: 'flex', gap: '10px', m: 0, p: 0, listStyle: 'none' }}>
        <StepChip n={FIND} label={t`Game folder`} state={stateOf(FIND)} />
        <StepChip n={NEXUS} label={t`Nexus Mods`} state={stateOf(NEXUS)} />
        {hasLoader ? (
          <StepChip
            n={LOADER}
            label={step > LOADER ? t`${game.loader} installed` : t`Install ${game.loader}`}
            state={stateOf(LOADER)}
          />
        ) : null}
        <StepChip
          n={hasLoader ? PROFILE : LOADER}
          label={t`First profile`}
          state={stateOf(PROFILE)}
        />
      </Box>
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
            onContinue={() => setStep(signedIn ? afterNexus : NEXUS)}
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
        {step === PROFILE ? <ProfileStep game={id} /> : null}
      </Box>
    </Box>
  )
}
