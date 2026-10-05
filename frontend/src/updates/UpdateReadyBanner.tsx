import { useLingui } from '@lingui/react/macro'
import { Alert, Button } from '@mui/material'
import { RotateCcw } from 'lucide-react'
import { useMortarUpdate } from '../settings/updates.ts'
import { reportUnexpected } from '../toasts/report.ts'

export function UpdateReadyBanner() {
  const { t } = useLingui()
  const { phase, release, restart } = useMortarUpdate()
  if (phase !== 'ready') {
    return null
  }
  return (
    <Alert
      severity="info"
      sx={{
        '--wails-draggable': 'no-drag',
        borderRadius: 0,
        fontSize: 14,
        alignItems: 'center',
      }}
      action={
        <Button
          color="inherit"
          size="small"
          startIcon={<RotateCcw size={16} />}
          onClick={() => restart().catch(reportUnexpected)}
        >
          {t`Restart now`}
        </Button>
      }
    >
      {t`Update ready: applies when you close Mortar.`}
      {release?.version ? ` (${release.version})` : ''}
    </Alert>
  )
}
