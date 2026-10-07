import { useLingui } from '@lingui/react/macro'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { useOverride } from './avOverride.ts'
import { reportUnexpected } from './report.ts'

/** The confirm behind "Install anyway": the antivirus flagged a download and the player may overrule it. */
export function AvOverrideDialog() {
  const { t } = useLingui()
  const asking = useOverride((s) => s.asking)
  const close = useOverride((s) => s.close)
  const where = asking?.file ? t` in ${asking.file}` : ''
  return (
    <ConfirmDialog
      open={asking !== null}
      color="error"
      title={t`Install ${asking?.title ?? ''} anyway?`}
      body={t`Your antivirus (${asking?.scanner ?? ''}) reports ${asking?.name ?? ''}${where}. Mod loaders and their DLLs often trip heuristics, but a real detection can harm your computer. Install only a file you trust. Mortar records this choice in the profile's history.`}
      confirmLabel={t`Install anyway`}
      onCancel={close}
      onConfirm={() => {
        const run = asking?.confirm
        close()
        Promise.resolve(run?.()).catch(reportUnexpected)
      }}
    />
  )
}
