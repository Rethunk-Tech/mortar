import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { ClipboardPaste, FolderOpen } from 'lucide-react'
import { reportUnexpected } from '../toasts/report.ts'
import type { ImportFlow } from './useImportFlow.ts'

// Before a preview exists: a link to type or paste, or a file to choose. Parse errors show here.
export function ImportInput({ flow }: { flow: ImportFlow }) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, maxWidth: 640 }}>
      {flow.tab === 'file' ? (
        <>
          <Typography sx={{ fontSize: 14 }}>
            {t`Open a .mortar file someone sent you. You see what it holds before anything downloads.`}
          </Typography>
          <Box>
            <Button
              variant="contained"
              startIcon={<FolderOpen size={16} />}
              disabled={flow.busy}
              onClick={() => {
                flow.pick().catch(reportUnexpected)
              }}
            >
              {t`Choose a file…`}
            </Button>
          </Box>
        </>
      ) : (
        <>
          <Typography sx={{ fontSize: 14 }}>
            {t`Paste a Mortar link. You see what it holds before anything downloads.`}
          </Typography>
          <TextField
            multiline={true}
            minRows={3}
            fullWidth={true}
            label={t`Share link`}
            value={flow.text}
            onChange={(e) => flow.setText(e.target.value)}
            slotProps={{ root: { sx: { userSelect: 'text' } } }}
          />
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              variant="contained"
              disabled={flow.busy || flow.text.trim() === ''}
              onClick={() => {
                flow.previewLink(flow.text).catch(reportUnexpected)
              }}
            >
              {t`Preview`}
            </Button>
            <Button
              variant="outlined"
              color="inherit"
              startIcon={<ClipboardPaste size={16} />}
              disabled={flow.busy}
              onClick={() => {
                flow.paste().catch(reportUnexpected)
              }}
            >
              {t`Paste from clipboard`}
            </Button>
          </Box>
        </>
      )}
      {flow.error ? (
        <Typography role="alert" sx={{ fontSize: 14, color: 'error.light' }}>
          {flow.error}
        </Typography>
      ) : null}
    </Box>
  )
}
