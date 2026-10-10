import { useLingui } from '@lingui/react/macro'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { useHandoffAsk } from './handoffConfirm.ts'

/** The confirm behind a file saved from an itch.io or Patreon page Mortar opened: record it against the page, or add it as a local file. */
export function HandoffDialog() {
  const { t } = useLingui()
  const asking = useHandoffAsk((s) => s.queue[0])
  const shift = useHandoffAsk((s) => s.shift)
  const answer = (yes: boolean) => {
    const a = asking
    shift()
    a?.answer(yes)
  }
  return (
    <ConfirmDialog
      open={asking !== undefined}
      title={t`Install ${asking?.file ?? ''} as from ${asking?.page ?? ''}?`}
      body={t`Mortar records the file against that page, so the mod's page link and shares point there. Choose No to add it as a plain local file.`}
      confirmLabel={t`Yes`}
      cancelLabel={t`No`}
      onCancel={() => answer(false)}
      onConfirm={() => answer(true)}
    />
  )
}
