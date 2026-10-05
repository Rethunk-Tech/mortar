import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  Typography,
} from '@mui/material'
import { useState } from 'react'
import type {
  Preview,
  Source,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/models.ts'
import {
  Import,
  Preview as PreviewPack,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import { PickPackFile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { packageLine } from './packImport.ts'

// Imports an r2modman or Gale profile code, or a profile export file, into a new profile: the code or file is read and
// listed first, and nothing is queued until Import.
export function PackImportDialog({
  open,
  game,
  onClose,
}: {
  open: boolean
  game: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const [text, setText] = useState('')
  const [path, setPath] = useState('')
  const [preview, setPreview] = useState<Preview | null>(null)
  const [busy, setBusy] = useState(false)
  const source = (): Source => ({ path, text: text.trim() })
  const close = () => {
    setText('')
    setPath('')
    setPreview(null)
    onClose()
  }
  const read = () => {
    setBusy(true)
    PreviewPack(source())
      .then(setPreview)
      .catch((error: unknown) => toastError(t`Could not read that code or file`, error))
      .finally(() => setBusy(false))
  }
  const importPack = () => {
    setBusy(true)
    Import(source(), game, '')
      .then((result) => {
        const unsupported = result.unsupported ?? []
        useToasts.getState().push({
          kind: unsupported.length > 0 ? 'warning' : 'success',
          title: t`Imported ${preview?.name ?? ''}: ${result.queued} to install`,
          ...(unsupported.length > 0
            ? { body: t`Not from Thunderstore: ${unsupported.join(', ')}` }
            : {}),
        })
        close()
      })
      .catch((error: unknown) => toastError(t`Could not import`, error))
      .finally(() => setBusy(false))
  }
  const packages = preview?.packages ?? []
  return (
    <Dialog open={open} onClose={close} fullWidth={true} maxWidth="xs">
      <DialogTitle>{t`Import from r2modman / Thunderstore`}</DialogTitle>
      <DialogContent dividers={true}>
        {preview ? (
          <Box>
            <Typography sx={{ fontWeight: 700 }}>{preview.name}</Typography>
            {packages.map((p) => (
              <Typography key={p.native} sx={{ fontSize: 14 }}>
                {packageLine(p, t`off`)}
              </Typography>
            ))}
            {preview.configs > 0 ? (
              <Typography color="text.secondary" sx={{ fontSize: 14, mt: 1 }}>
                {t`${preview.configs} config files come with it.`}
              </Typography>
            ) : null}
          </Box>
        ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <TextField
              fullWidth={true}
              size="small"
              label={t`Profile code`}
              value={text}
              onChange={(event) => setText(event.target.value)}
            />
            <Button
              onClick={() => {
                PickPackFile()
                  .then((picked) => picked && setPath(picked))
                  .catch((error: unknown) => toastError(t`Could not open the file`, error))
              }}
            >
              {path || t`Choose a file…`}
            </Button>
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={close}>{t`Cancel`}</Button>
        {preview ? (
          <Button variant="contained" disabled={busy} onClick={importPack}>
            {t`Import`}
          </Button>
        ) : (
          <Button
            variant="contained"
            disabled={busy || (text.trim() === '' && path === '')}
            onClick={read}
          >
            {t`Preview`}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  )
}
