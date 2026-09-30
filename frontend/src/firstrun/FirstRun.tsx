import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import { type GameStatus, loadGameStatus } from '../games/status.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { FindStep } from './FindStep.tsx'
import { STARDEW } from './needed.ts'
import { ProfileStep } from './ProfileStep.tsx'
import { SmapiStep } from './SmapiStep.tsx'

const FIND = 1
const SMAPI = 2
const PROFILE = 3
type Step = typeof FIND | typeof SMAPI | typeof PROFILE

function Chip({ n, label, state }: { n: Step; label: string; state: 'done' | 'current' | 'todo' }) {
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

export function FirstRun() {
  const { t } = useLingui()
  const [step, setStep] = useState<Step>(FIND)
  const [status, setStatus] = useState<GameStatus | null>(null)
  const refresh = useCallback(() => {
    loadGameStatus().then(setStatus).catch(reportUnexpected)
  }, [])
  useEffect(refresh, [refresh])
  const goToProfile = useCallback(() => setStep(PROFILE), [])

  if (!status) {
    return null
  }
  const stateOf = (n: Step) => {
    if (n < step) {
      return 'done'
    }
    return n === step ? 'current' : 'todo'
  }
  const gameDir = status.games.find((g) => g.id === STARDEW)?.installDir ?? ''
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
      <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '6px' }}>
        <Typography component="h1" sx={{ fontSize: 40, fontWeight: 700 }}>
          {t`Welcome to Mortar`}
        </Typography>
        <Typography
          sx={{ fontSize: 17 }}
        >{t`Three steps and you are playing with mods.`}</Typography>
      </Box>
      <Box component="ol" sx={{ display: 'flex', gap: '10px', m: 0, p: 0, listStyle: 'none' }}>
        <Chip
          n={FIND}
          label={step > FIND ? t`Game found` : t`Find the game`}
          state={stateOf(FIND)}
        />
        <Chip
          n={SMAPI}
          label={step > SMAPI ? t`SMAPI installed` : t`Install SMAPI`}
          state={stateOf(SMAPI)}
        />
        <Chip n={PROFILE} label={t`First profile`} state={stateOf(PROFILE)} />
      </Box>
      {step === FIND ? (
        <FindStep status={status} refresh={refresh} onContinue={() => setStep(SMAPI)} />
      ) : null}
      {step === SMAPI ? <SmapiStep gameDir={gameDir} onDone={goToProfile} /> : null}
      {step === PROFILE ? <ProfileStep /> : null}
    </Box>
  )
}
