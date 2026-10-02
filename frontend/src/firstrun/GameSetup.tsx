import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { type ReactNode, useCallback, useEffect, useState } from 'react'
import type {
  GameInfo,
  StoreApp,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import {
  Launchers,
  List,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/service.ts'
import type { GameId } from '../nav/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { FindStep } from './FindStep.tsx'
import { ProfileStep } from './ProfileStep.tsx'
import { SmapiStep } from './SmapiStep.tsx'

const FIND = 1
const LOADER = 2
const PROFILE = 3
type Step = typeof FIND | typeof LOADER | typeof PROFILE

type LoaderStep = (p: { game: GameId; gameDir: string; onDone: () => void }) => ReactNode

// Each game's own setup between finding its folder and its first profile; a game without a loader has none.
const loaderSteps: Record<GameId, LoaderStep | null> = {
  stardew: SmapiStep,
}

export function StepChip({
  n,
  label,
  state,
}: {
  n: number
  label: string
  state: 'done' | 'current' | 'todo'
}) {
  return (
    <Box
      component="li"
      aria-current={state === 'current' ? 'step' : undefined}
      sx={{
        px: '14px',
        py: '6px',
        borderRadius: '16px',
        fontSize: 14,
        fontWeight: state === 'todo' ? 'normal' : 'bold',
        whiteSpace: 'nowrap',
        bgcolor: (theme) =>
          ({
            done: 'rgba(12,223,100,0.18)',
            current: theme.palette.primary.main,
            todo: 'rgba(255,255,255,0.1)',
          })[state],
        color: (theme) =>
          ({
            done: theme.palette.success.light,
            current: theme.palette.primary.contrastText,
            todo: theme.palette.text.secondary,
          })[state],
      }}
    >
      {n} {label}
    </Box>
  )
}

// Sets up one game the first time it is opened: its folder, its loader, then a first profile.
export function GameSetup({ game: id }: { game: GameId }) {
  const { t } = useLingui()
  const [step, setStep] = useState<Step>(FIND)
  const [game, setGame] = useState<GameInfo | null>(null)
  const [launchers, setLaunchers] = useState<StoreApp[]>([])
  const refresh = useCallback(() => {
    Promise.all([List(), Launchers()])
      .then(([games, ls]) => {
        setGame((games ?? []).find((g) => g.id === id) ?? null)
        setLaunchers(ls ?? [])
      })
      .catch(reportUnexpected)
  }, [id])
  useEffect(refresh, [refresh])
  const goToProfile = useCallback(() => setStep(PROFILE), [])
  const Loader = loaderSteps[id]

  if (!game) {
    return null
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
        {Loader ? (
          <StepChip
            n={LOADER}
            label={step > LOADER ? t`${game.loader} installed` : t`Install ${game.loader}`}
            state={stateOf(LOADER)}
          />
        ) : null}
        <StepChip n={Loader ? PROFILE : LOADER} label={t`First profile`} state={stateOf(PROFILE)} />
      </Box>
      {step === FIND ? (
        <FindStep
          game={game}
          launchers={launchers}
          refresh={refresh}
          onContinue={() => setStep(Loader ? LOADER : PROFILE)}
        />
      ) : null}
      {step === LOADER && Loader ? (
        <Loader game={id} gameDir={game.installDir} onDone={goToProfile} />
      ) : null}
      {step === PROFILE ? <ProfileStep game={id} /> : null}
    </Box>
  )
}
