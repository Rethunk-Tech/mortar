import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Typography } from '@mui/material'
import {
  Cancel,
  ConfirmLarge,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { formatKb } from '../i18n/bytes.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { largeAsk } from './large.ts'
import { useQueue } from './store.ts'

/**
 * Asks before a download over 2 GB starts. Cancel takes focus and closing the dialog cancels too, so only a
 * deliberate click on "Continue anyway" lets it through.
 */
export function LargeDownloadDialog() {
  const { t } = useLingui()
  const ask = largeAsk(useQueue((s) => s.state.items))
  const item = ask?.item
  const name = item?.name || item?.fileName || item?.package || ''
  const size = formatKb(item?.sizeKb ?? 0)
  const more = ask?.more ?? 0
  const cancel = () => {
    if (item) {
      Promise.resolve(Cancel(item.id)).catch(reportUnexpected)
    }
  }
  return (
    <ConfirmationDialog
      open={ask !== null}
      tone="danger"
      title={t`${name} is a very large download`}
      body={t`This download is ${size}, far larger than a mod normally is. It will take a long time and fill a lot of your drive. Make sure you meant to get it.`}
      confirmLabel={t`Continue anyway`}
      onCancel={cancel}
      onConfirm={() => {
        if (item) {
          Promise.resolve(ConfirmLarge(item.id)).catch(reportUnexpected)
        }
      }}
    >
      {more > 0 ? (
        <Typography sx={{ color: 'text.secondary' }}>
          {t`${plural(more, { one: '# more large download is waiting after this one.', other: '# more large downloads are waiting after this one.' })}`}
        </Typography>
      ) : null}
    </ConfirmationDialog>
  )
}
