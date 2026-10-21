import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { ClipboardPaste, FolderOpen } from 'lucide-react'
import { liveHandoff, useHandoff } from '../install/handoff.ts'
import { space } from '../theme/density.ts'
import { BUTTON_LARGE_HEIGHT } from '../theme/theme.ts'
import { reportUnexpected } from '../toasts/report.ts'
import type { ImportFlow } from './useImportFlow.ts'

// The buttons beside the link field keep their one-line labels and the field's height.
const rowButtonSx = { flexShrink: 0, whiteSpace: 'nowrap', height: BUTTON_LARGE_HEIGHT } as const

// Before a preview exists: a link to type or paste, or a file to choose. Parse errors show here.
export function ImportInput({ flow }: { flow: ImportFlow }) {
  const { t } = useLingui()
  const handoff = liveHandoff(useHandoff((s) => s.page))
  const canPreview = !flow.busy && flow.text.trim() !== ''
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap, p: '20px 18px' }}>
      {flow.tab === 'file' ? (
        <>
          <Typography sx={{ fontSize: 14, textAlign: 'center' }}>
            {t`Open a .mortar file, an r2modman or Gale .r2z, or a Thunderstore modpack. You see what it holds before anything downloads.`}
          </Typography>
          <Box sx={{ display: 'flex', justifyContent: 'center' }}>
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
            {t`Paste a Mortar link, a Nexus collection, an r2modman or Gale code, a Patreon post, an itch.io page or (Premium) a Nexus mod link.`}
          </Typography>
          <Box
            component="form"
            onSubmit={(e) => {
              e.preventDefault()
              if (canPreview) {
                flow.previewLink(flow.text).catch(reportUnexpected)
              }
            }}
            sx={{ display: 'flex', gap: space.gap }}
          >
            <TextField
              size="small"
              fullWidth={true}
              autoFocus={true}
              placeholder={t`Link or code`}
              value={flow.text}
              onChange={(e) => flow.setText(e.target.value)}
              slotProps={{
                htmlInput: { 'aria-label': t`Link or code`, spellCheck: false },
              }}
              // As tall as the large button beside it.
              sx={{ userSelect: 'text', '& .MuiInputBase-root': { height: BUTTON_LARGE_HEIGHT } }}
            />
            <Button
              variant="outlined"
              color="inherit"
              startIcon={<ClipboardPaste size={16} />}
              disabled={flow.busy}
              onClick={() => {
                flow.paste().catch(reportUnexpected)
              }}
              size="large"
              sx={rowButtonSx}
            >
              {t`Paste`}
            </Button>
            <Button
              type="submit"
              variant="contained"
              disabled={!canPreview}
              size="large"
              sx={rowButtonSx}
            >
              {t`Preview`}
            </Button>
          </Box>
        </>
      )}
      {handoff ? (
        <Typography role="status" sx={{ fontSize: 14, color: 'text.secondary' }}>
          {handoff.kind === 'patreon'
            ? t`Opened the Patreon post. Save the file from it; Mortar offers it from your Downloads folder.`
            : t`Opened the itch.io page. Save the file from it; Mortar offers it from your Downloads folder.`}
        </Typography>
      ) : null}
      {flow.error ? (
        <Typography role="alert" sx={{ fontSize: 14, color: 'error.light' }}>
          {flow.error}
        </Typography>
      ) : null}
    </Box>
  )
}
