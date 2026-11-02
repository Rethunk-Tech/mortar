import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { reportUnexpected } from '../toasts/report.ts'
import { useLaunch } from './store.ts'

export function PlayAnywayButton() {
  const { t } = useLingui()
  const playAnyway = useLaunch((s) => s.playAnyway)
  return (
    <Button variant="contained" onClick={() => playAnyway().catch(reportUnexpected)}>
      {t`Play anyway`}
    </Button>
  )
}
