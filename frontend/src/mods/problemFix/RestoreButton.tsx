import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { reportUnexpected } from '../../toasts/report.ts'
import { useMods } from '../store.ts'

export function RestoreButton({ token }: { token: string }) {
  const { t } = useLingui()
  const restoreDismissed = useMods((s) => s.restoreDismissed)
  return (
    <Button
      size="small"
      color="inherit"
      variant="text"
      onClick={() => restoreDismissed(token).catch(reportUnexpected)}
      sx={{ flexShrink: 0 }}
    >
      {t`Restore`}
    </Button>
  )
}
