import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  ClickAwayListener,
  Fade,
  Paper,
  Popper,
  Stack,
  Typography,
} from '@mui/material'
import { useId } from 'react'
import { mergeBindings } from '../settings/shortcuts.ts'
import { useSettings } from '../settings/store.ts'
import { TOUR_STEP_COUNT, tourOnLastStep, tourStepBack, tourStepNext } from './logic.ts'
import {
  TOUR_STEP_COMMAND,
  TOUR_STEP_MODS,
  TOUR_STEP_PLAY,
  TOUR_STEP_PROBLEMS,
  TOUR_STEP_PROFILES,
} from './steps.ts'

const TOUR_Z_INDEX = 1400
const POPPER_OFFSET = 12

const placements = ['right-start', 'right', 'bottom', 'bottom', 'top'] as const

function TourPopover({
  anchorEl,
  step,
  setStep,
  finish,
}: {
  anchorEl: HTMLElement
  step: number
  setStep: (value: number | ((prev: number) => number)) => void
  finish: () => void
}) {
  const { t } = useLingui()
  const titleId = useId()
  const shortcuts = useSettings((s) => s.shortcuts)
  const paletteKeys = mergeBindings(shortcuts)['command-palette']
  const placement = placements[step] ?? 'bottom'
  const last = tourOnLastStep(step)

  let title = t`Profiles`
  let body = t`Switch mod sets here. Right-click a profile for rename, duplicate, and more.`
  if (step === TOUR_STEP_PLAY) {
    title = t`Play`
    body = t`Launch the game with this profile's mods. Mortar applies your list before SMAPI starts.`
  } else if (step === TOUR_STEP_MODS) {
    title = t`Mods`
    body = t`Enable, disable, and update mods for the open profile.`
  } else if (step === TOUR_STEP_PROBLEMS) {
    title = t`Problems`
    body = t`See load errors, missing dependencies, and other issues before you play.`
  } else if (step === TOUR_STEP_COMMAND) {
    title = t`Command palette`
    body = t`Press ${paletteKeys} to jump anywhere — profiles, settings, downloads, and more.`
  }

  const stepLabel = t`Step ${step + 1} of ${TOUR_STEP_COUNT}`

  return (
    <>
      <Box
        aria-hidden={true}
        sx={{
          position: 'fixed',
          inset: 0,
          zIndex: TOUR_Z_INDEX - 1,
          bgcolor: 'rgba(0,0,0,0.35)',
          pointerEvents: 'none',
        }}
      />
      <Popper
        open={true}
        anchorEl={anchorEl}
        placement={placement}
        transition={true}
        modifiers={[{ name: 'offset', options: { offset: [0, POPPER_OFFSET] } }]}
        sx={{ zIndex: TOUR_Z_INDEX }}
      >
        {({ TransitionProps }) => (
          <Fade {...TransitionProps} timeout={200}>
            <Paper
              role="dialog"
              aria-labelledby={titleId}
              elevation={8}
              sx={{
                maxWidth: 320,
                p: 2,
                border: '1px solid var(--mortar-hairline-12)',
                bgcolor: 'var(--mortar-panel-solid)',
              }}
            >
              <ClickAwayListener onClickAway={() => undefined}>
                <Stack spacing={1.5}>
                  <Typography id={titleId} sx={{ fontSize: 15, fontWeight: 700 }}>
                    {title}
                  </Typography>
                  <Typography sx={{ fontSize: 13, lineHeight: 1.5, color: 'text.secondary' }}>
                    {body}
                  </Typography>
                  <Typography sx={{ fontSize: 12, color: 'text.disabled' }}>{stepLabel}</Typography>
                  <Stack
                    direction="row"
                    spacing={1}
                    justifyContent="space-between"
                    alignItems="center"
                  >
                    <Button size="small" color="inherit" onClick={finish}>
                      {t`Skip tour`}
                    </Button>
                    <Stack direction="row" spacing={1}>
                      <Button
                        size="small"
                        disabled={step === TOUR_STEP_PROFILES}
                        onClick={() => setStep((s) => tourStepBack(s))}
                      >
                        {t`Back`}
                      </Button>
                      <Button
                        size="small"
                        variant="contained"
                        onClick={() => {
                          if (last) {
                            finish()
                          } else {
                            setStep((s) => tourStepNext(s))
                          }
                        }}
                      >
                        {last ? t`Done` : t`Next`}
                      </Button>
                    </Stack>
                  </Stack>
                </Stack>
              </ClickAwayListener>
            </Paper>
          </Fade>
        )}
      </Popper>
    </>
  )
}

export { TourPopover }
