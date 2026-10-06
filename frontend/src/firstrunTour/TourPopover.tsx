import { useLingui } from '@lingui/react/macro'
import { Box, Button, Fade, Paper, Popper, Stack, Typography } from '@mui/material'
import { useEffect, useId, useRef } from 'react'
import { mergeBindings } from '../settings/shortcuts.ts'
import { useSettings } from '../settings/store.ts'
import { controlsCutout } from '../shell/controlsCutout.ts'
import {
  TOUR_STEP_COUNT,
  type TourRect,
  tourOnLastStep,
  tourStepBack,
  tourStepNext,
} from './logic.ts'
import { TOUR_STEPS } from './steps.ts'

const TOUR_Z_INDEX = 1400
const POPPER_OFFSET = 12
const SPOTLIGHT_PAD = 6

const placements = [
  'right-start',
  'bottom',
  'bottom',
  'right',
  'bottom',
  'bottom',
  'bottom',
  'bottom',
  'bottom-start',
] as const

function useTourCopy(step: number, paletteKeys: string) {
  const { t } = useLingui()
  let title = t`Profiles`
  let body = t`Switch mod sets here. Right-click a profile for rename, duplicate, and more.`
  switch (TOUR_STEPS[step]) {
    case 'browse':
      title = t`Browse`
      body = t`Search every mod site at once or pick one; with nothing typed you see the top mods. Add installs into this profile. You can also drop a downloaded archive anywhere on this window.`
      break
    case 'game':
      title = t`Switch game`
      body = t`Click the game's name in the title bar to open the game drawer and jump to another game.`
      break
    case 'play':
      title = t`Play`
      body = t`Launch the game with this profile's mods. Mortar applies your list before the game starts.`
      break
    case 'mods':
      title = t`Mods`
      body = t`Enable and disable mods, and update them for the open profile.`
      break
    case 'details':
      title = t`Mod details`
      body = t`Click a mod to open its details: versions, what it needs and what needs it, and its update.`
      break
    case 'config':
      title = t`Mod config`
      body = t`A mod with a config file has an Edit config button in its details: a typed editor with presets and Reset all.`
      break
    case 'problems':
      title = t`Problems`
      body = t`See load errors, missing dependencies, and other issues before you play.`
      break
    case 'command':
      title = t`Command palette`
      body = t`Press ${paletteKeys} to jump anywhere: profiles, settings, downloads and more.`
      break
    default:
      break
  }
  return { title, body }
}

function TourPopover({
  anchorEl,
  anchorRect,
  step,
  setStep,
  finish,
}: {
  anchorEl: HTMLElement
  anchorRect: TourRect
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
  const previousFocus = useRef<HTMLElement | null>(null)
  useEffect(() => {
    const el = document.activeElement
    previousFocus.current = el instanceof HTMLElement ? el : null
  }, [])
  const close = () => {
    finish()
    previousFocus.current?.focus()
  }

  const { title, body } = useTourCopy(step, paletteKeys)

  const stepLabel = t`Step ${step + 1} of ${TOUR_STEP_COUNT}`

  return (
    <>
      <Box
        aria-hidden={true}
        sx={{
          position: 'fixed',
          inset: 0,
          zIndex: TOUR_Z_INDEX - 1,
          pointerEvents: 'none',
          clipPath: controlsCutout,
        }}
      >
        <Box
          data-tour-spotlight={true}
          sx={{
            position: 'fixed',
            top: anchorRect.top - SPOTLIGHT_PAD,
            left: anchorRect.left - SPOTLIGHT_PAD,
            width: anchorRect.width + SPOTLIGHT_PAD * 2,
            height: anchorRect.height + SPOTLIGHT_PAD * 2,
            borderRadius: 1,
            outline: '2px solid',
            outlineColor: 'primary.main',
            // The shadow dims everything outside the anchor, leaving the anchor itself lit.
            boxShadow: '0 0 0 100vmax var(--mortar-scrim)',
            transition: 'top 200ms, left 200ms, width 200ms, height 200ms',
          }}
        />
      </Box>
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
              onKeyDown={(e) => {
                if (e.key === 'Escape') {
                  e.stopPropagation()
                  close()
                }
              }}
              elevation={8}
              sx={{
                maxWidth: 320,
                p: 2,
                border: '1px solid var(--mortar-hairline-12)',
                bgcolor: 'var(--mortar-panel-solid)',
              }}
            >
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
                  sx={{ justifyContent: 'space-between', alignItems: 'center' }}
                >
                  <Button size="small" color="inherit" onClick={close}>
                    {t`Skip tour`}
                  </Button>
                  <Stack direction="row" spacing={1}>
                    <Button
                      size="small"
                      disabled={step === 0}
                      onClick={() => setStep((s) => tourStepBack(s))}
                    >
                      {t`Back`}
                    </Button>
                    <Button
                      size="small"
                      variant="contained"
                      autoFocus={true}
                      onClick={() => {
                        if (last) {
                          close()
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
            </Paper>
          </Fade>
        )}
      </Popper>
    </>
  )
}

export { TourPopover }
