import { useLingui } from '@lingui/react/macro'
import { useProfiles } from '../profiles/store.ts'
import { PromptDialog } from '../shell/PromptDialog.tsx'
import { usePending } from '../toasts/usePending.ts'

export function NewProfileDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const create = useProfiles((s) => s.create)
  const [busy, run] = usePending()
  return (
    <PromptDialog
      open={open}
      title={t`New profile`}
      label={t`Profile name`}
      maxLength={60}
      confirmLabel={t`Create`}
      busy={busy}
      onCancel={onClose}
      onSubmit={(name) => {
        run(() => create(name).then(() => onClose()), {
          errorTitle: t`Could not create the profile`,
        })
      }}
    />
  )
}
