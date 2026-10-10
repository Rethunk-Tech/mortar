import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { ClipboardPaste, FolderOpen } from 'lucide-react'
import { liveItchPage, useItchPage } from '../install/itch.ts'
import { livePatreonPost, usePatreonPost } from '../install/patreon.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import type { ImportFlow } from './useImportFlow.ts'

// Before a preview exists: a link to type or paste, or a file to choose. Parse errors show here.
export function ImportInput({ flow }: { flow: ImportFlow }) {
  const { t } = useLingui()
  const game = useCurrentGame()
  const patreonPost = livePatreonPost(usePatreonPost((s) => s.post))
  const itchPage = liveItchPage(useItchPage((s) => s.page))
  const canPreview = !flow.busy && flow.text.trim() !== ''
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap, p: '20px 18px' }}>
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
            {t`Paste a share link, a collection link, a Patreon post link, an itch.io page link, or (Premium) a Nexus mod link with Ctrl+V.`}
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
              placeholder={`https://mortar.rethunk.tech/${game}/p#…`}
              value={flow.text}
              onChange={(e) => flow.setText(e.target.value)}
              slotProps={{
                htmlInput: { 'aria-label': t`Share link`, spellCheck: false },
              }}
              sx={{ userSelect: 'text', '& .MuiInputBase-root': { height: 44 } }}
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
            >
              {t`Paste from clipboard`}
            </Button>
            <Button type="submit" variant="contained" disabled={!canPreview} size="large">
              {t`Preview`}
            </Button>
          </Box>
        </>
      )}
      {patreonPost ? (
        <Typography role="status" sx={{ fontSize: 14, color: 'text.secondary' }}>
          {t`Opened the Patreon post. Save the file from it; Mortar offers it from your Downloads folder.`}
        </Typography>
      ) : null}
      {itchPage ? (
        <Typography role="status" sx={{ fontSize: 14, color: 'text.secondary' }}>
          {t`Opened the itch.io page. Save the file from it; Mortar offers it from your Downloads folder.`}
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
