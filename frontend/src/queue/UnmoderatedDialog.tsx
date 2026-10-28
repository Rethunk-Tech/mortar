import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { listNames } from '../i18n/list.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { useUnmoderatedAsk } from './unmoderated.ts'

/** The one-time warning before the first download from a site nobody reviews uploads on. */
export function UnmoderatedDialog() {
  const { t } = useLingui()
  const ask = useUnmoderatedAsk((s) => s.ask)
  const sites = listNames((ask?.sources ?? []).map(sourceLabel))
  const count = ask?.sources.length ?? 0
  return (
    <ConfirmationDialog
      open={ask !== null}
      title={t`${plural(count, { one: `${{ sites }} is not moderated`, other: `${{ sites }} are not moderated` })}`}
      body={t`Anyone can publish files on ${sites}, and nobody reviews or scans them for malware. Download only from authors you trust. Mortar asks once per site.`}
      confirmLabel={t`Download`}
      onCancel={() => ask?.answer(false)}
      onConfirm={() => ask?.answer(true)}
    />
  )
}
