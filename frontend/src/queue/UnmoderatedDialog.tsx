import { useLingui } from '@lingui/react/macro'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { listNames } from '../i18n/list.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { useUnmoderatedAsk } from './unmoderated.ts'

/** The one-time warning before the first download from a site nobody reviews uploads on. */
export function UnmoderatedDialog() {
  const { t } = useLingui()
  const ask = useUnmoderatedAsk((s) => s.ask)
  const sites = listNames((ask?.sources ?? []).map(sourceLabel))
  return (
    <ConfirmDialog
      open={ask !== null}
      title={t`${sites} is not moderated`}
      body={t`Anyone can publish files on ${sites}, and nobody reviews or scans them for malware. Download only from authors you trust. Mortar asks once per site.`}
      confirmLabel={t`Download`}
      onCancel={() => ask?.answer(false)}
      onConfirm={() => ask?.answer(true)}
    />
  )
}
