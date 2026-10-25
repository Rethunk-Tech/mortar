import { useLingui } from '@lingui/react/macro'
import { type ReactNode, useState } from 'react'
import { ConfirmationDialog } from './dialog/ConfirmationDialog.tsx'

/** Routes a form dialog's dismissal through "Discard changes?" when anything was edited. */
export function useDiscardGuard(
  dirty: boolean,
  onClose: () => void,
): { request: () => void; dialog: ReactNode } {
  const { t } = useLingui()
  const [asking, setAsking] = useState(false)
  const dialog = (
    <ConfirmationDialog
      open={asking}
      title={t`Discard changes?`}
      confirmLabel={t`Discard`}
      tone="danger"
      onCancel={() => setAsking(false)}
      onConfirm={() => {
        setAsking(false)
        onClose()
      }}
    />
  )
  return { request: () => (dirty ? setAsking(true) : onClose()), dialog }
}
