import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { useState } from 'react'
import { CancelLaunch } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { useTab } from '../game/tab.ts'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { type Phase, useLaunchView } from './progress.ts'
import { useLaunch } from './store.ts'

/** The launch screen's buttons: Cancel (Close once it failed), and Open console, or Hide when the Console is open. */
export function Buttons({ game, phase, focus }: { game: string; phase: Phase; focus: boolean }) {
  const { t } = useLingui()
  const hide = useLaunch((s) => s.hide)
  const dismissFailure = useLaunch((s) => s.dismissFailure)
  const close = useLaunchView((s) => s.close)
  const setTab = useTab((s) => s.setTab)
  // On the Console already, Open console would only hide the screen, so Hide stands in for it.
  const onConsole = useTab((s) => s.tab === 'console')
  const failed = phase === 'failed'
  const closeFailure = () => {
    dismissFailure()
    close()
  }
  // Ending a game takes a moment, so Cancel says it was heard until the screen closes.
  const [cancelling, setCancelling] = useState(false)
  const cancel = () => {
    setCancelling(true)
    CancelLaunch(game).catch((e: unknown) => {
      setCancelling(false)
      reportError(t`Could not cancel the launch`)(e)
    })
  }
  return (
    <Box
      sx={{
        display: 'flex',
        gap: space.gap,
        visibility: phase === 'running' ? 'hidden' : 'visible',
      }}
    >
      {failed ? (
        <Button onClick={closeFailure}>{t`Close`}</Button>
      ) : (
        <Button onClick={cancel} disabled={cancelling}>
          {cancelling ? t`Cancelling…` : t`Cancel`}
        </Button>
      )}
      {onConsole ? null : (
        <Button
          variant="outlined"
          autoFocus={focus}
          onClick={() => {
            setTab('console')
            if (failed) {
              closeFailure()
            } else {
              hide()
            }
          }}
        >
          {t`Open console`}
        </Button>
      )}
      {onConsole && !failed ? (
        <Button variant="outlined" autoFocus={focus} onClick={hide}>
          {t`Hide`}
        </Button>
      ) : null}
    </Box>
  )
}
