import { useLingui } from '@lingui/react/macro'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { usePending } from '../toasts/usePending.ts'

/** Asks before a crash check, which starts the game over and over on a copy of the profile. */
export function BisectConfirm({
  open,
  onCancel,
  onConfirm,
}: {
  open: boolean
  onCancel: () => void
  onConfirm: () => Promise<void>
}) {
  const { t } = useLingui()
  const [busy, run] = usePending()
  return (
    <ConfirmationDialog
      open={open}
      title={t`Find the mod that crashes the game?`}
      body={t`Mortar starts the game on a copy of this profile, turning half the mods off each time, until one mod is left. Your profile stays as it is.`}
      confirmLabel={t`Start`}
      busy={busy}
      onCancel={onCancel}
      onConfirm={() => run(onConfirm, { errorTitle: t`Could not start the crash check` })}
    />
  )
}
