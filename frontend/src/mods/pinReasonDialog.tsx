import { useLingui } from '@lingui/react/macro'
import { PromptDialog } from '../shell/PromptDialog.tsx'
import { usePending } from '../toasts/usePending.ts'
import { usePinReasonDialog } from './pinReasonStore.ts'
import { pinMany, pinMod } from './storeEntries.ts'

export function PinReasonDialog() {
  const { t } = useLingui()
  const target = usePinReasonDialog((s) => s.target)
  const close = usePinReasonDialog((s) => s.close)
  const [pending, run] = usePending()
  const open = target !== null
  const name = target?.mods[0]?.name ?? ''
  const many = (target?.mods.length ?? 0) > 1
  return (
    <PromptDialog
      open={open}
      title={many ? t`Pin selected versions` : t`Pin ${name}`}
      label={t`Reason (optional)`}
      allowEmpty={true}
      busy={pending}
      confirmLabel={t`Pin`}
      onCancel={() => {
        if (!pending) {
          close()
        }
      }}
      onSubmit={(reason) => {
        if (!target) {
          return
        }
        run(async () => {
          const [only] = target.mods
          if (target.mods.length === 1 && only) {
            await pinMod(only, true, reason)
          } else {
            await pinMany(target.mods, true, reason)
          }
          close()
        })
      }}
    />
  )
}
