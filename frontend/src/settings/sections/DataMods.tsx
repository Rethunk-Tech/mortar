import { useLingui } from '@lingui/react/macro'
import { ClearCache } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/service.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'

function CacheClearDialog({
  open,
  onClose,
  onCleared,
}: {
  open: boolean
  onClose: () => void
  onCleared: () => void
}) {
  const { t } = useLingui()
  return (
    <ConfirmDialog
      open={open}
      title={t`Clear cache?`}
      body={t`Removes cached Nexus and SMAPI details and problem scans. Use it if an API key was revoked or the details look wrong. Mortar fetches them again as needed, so the next problem check takes longer.`}
      confirmLabel={t`Clear cache`}
      onCancel={onClose}
      onConfirm={() => {
        ClearCache()
          .then(() => {
            onClose()
            onCleared()
            useToasts.getState().push({ kind: 'success', title: t`Cache cleared` })
          })
          .catch(reportUnexpected)
      }}
    />
  )
}

export { CacheClearDialog }
