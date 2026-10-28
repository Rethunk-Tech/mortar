import { useLingui } from '@lingui/react/macro'
import { Typography } from '@mui/material'
import { useState } from 'react'
import { historyLabel } from '../../profiles/historyLabel.ts'
import { ConfirmationDialog } from '../../shell/dialog/ConfirmationDialog.tsx'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { restoreBatch, useUndoAll } from './undoAll.ts'

/** Undo all after the profile changed again: lists what restoring would revert besides the batch. */
export function UndoAllConfirm() {
  const { t } = useLingui()
  const target = useUndoAll((s) => s.target)
  const [busy, setBusy] = useState(false)
  const close = () => useUndoAll.setState({ target: null })
  return (
    <ConfirmationDialog
      open={target !== null}
      title={t`Undo all also reverts later changes`}
      confirmLabel={t`Undo all`}
      tone="warning"
      busy={busy}
      onCancel={close}
      onConfirm={() => {
        if (!target) {
          return
        }
        setBusy(true)
        restoreBatch(target.game, target.profileId, target.beforeId)
          .catch(reportUnexpected)
          .finally(() => {
            setBusy(false)
            close()
          })
      }}
    >
      <Typography sx={{ fontSize: 'base.fontSize', mb: space.gap }}>
        {t`Restoring from before Update all also reverts these later changes:`}
      </Typography>
      {(target?.later ?? []).map((e) => (
        <Typography key={e.id} sx={{ fontSize: 'base.fontSize' }} color="text.secondary">
          {historyLabel(e)}
        </Typography>
      ))}
    </ConfirmationDialog>
  )
}
