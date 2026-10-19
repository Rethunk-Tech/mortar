import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@mui/material'
import { TriangleAlert } from 'lucide-react'
import {
  Cancel,
  ConfirmLarge,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { formatKb } from '../i18n/bytes.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { largeAsk } from './large.ts'
import { useQueue } from './store.ts'

/**
 * Asks before a download over 2 GB starts. Cancel is the filled, focused button and closing the dialog cancels too,
 * so only a deliberate click on "Continue anyway" lets it through.
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
    <Dialog
      open={ask !== null}
      onClose={cancel}
      slotProps={{
        paper: { sx: { maxWidth: 460, borderTop: 4, borderColor: 'error.main' } },
      }}
    >
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1, color: 'error.main' }}>
        <TriangleAlert size={22} aria-hidden={true} />
        {t`${name} is a very large download`}
      </DialogTitle>
      <DialogContent>
        <DialogContentText>
          {t`This download is ${size}, far larger than a mod normally is. It will take a long time and fill a lot of your drive. Make sure you meant to get it.`}
        </DialogContentText>
        {more > 0 ? (
          <Box sx={{ mt: 1 }}>
            <DialogContentText>{t`${more} more large downloads are waiting after this one.`}</DialogContentText>
          </Box>
        ) : null}
      </DialogContent>
      <DialogActions sx={{ flexWrap: 'wrap' }}>
        <Button
          variant="outlined"
          color="error"
          onClick={() => {
            if (item) {
              Promise.resolve(ConfirmLarge(item.id)).catch(reportUnexpected)
            }
          }}
        >
          {t`Continue anyway`}
        </Button>
        <Button variant="contained" autoFocus={true} onClick={cancel}>
          {t`Cancel`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
