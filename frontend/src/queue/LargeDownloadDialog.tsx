import { useLingui } from '@lingui/react/macro'
import {
  Cancel,
  ConfirmLarge,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { formatKb } from '../i18n/bytes.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { largeAsk } from './large.ts'
import { useQueue } from './store.ts'

/** Asks before a download over 2 GB starts. Cancel is the default; only "Continue anyway" lets it through. */
export function LargeDownloadDialog() {
  const { t } = useLingui()
  const ask = largeAsk(useQueue((s) => s.state.items))
  const item = ask?.item
  const name = item?.name || item?.fileName || item?.package || ''
  const size = formatKb(item?.sizeKb ?? 0)
  const more = ask?.more ?? 0
  return (
    <ConfirmDialog
      open={ask !== null}
      color="warning"
      focusCancel={true}
      title={t`${name} is a very large download`}
      body={
        more > 0
          ? t`This download is ${size}, which is unusually large for a mod. It will take a long time and a lot of disk space. ${more} more large downloads are waiting after this one.`
          : t`This download is ${size}, which is unusually large for a mod. It will take a long time and a lot of disk space.`
      }
      confirmLabel={t`Continue anyway`}
      onCancel={() => {
        if (item) {
          Promise.resolve(Cancel(item.id)).catch(reportUnexpected)
        }
      }}
      onConfirm={() => {
        if (item) {
          Promise.resolve(ConfirmLarge(item.id)).catch(reportUnexpected)
        }
      }}
    />
  )
}
