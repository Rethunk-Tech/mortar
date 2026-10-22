import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, InputAdornment, TextField, Typography } from '@mui/material'
import { ClipboardPaste, FolderOpen } from 'lucide-react'
import { GALE, R2MODMAN } from '../brand/productNames.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { liveHandoff, useHandoff } from '../install/handoff.ts'
import { Tooltip } from '../shell/Tooltip.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import type { ImportFlow } from './useImportFlow.ts'

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
            {t`Open a .mortar file, an ${{ manager: R2MODMAN }} or ${{ otherManager: GALE }} .r2z, or a ${{ source: sourceLabel('thunderstore') }} modpack. You see what it holds before anything downloads.`}
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
          <Typography sx={{ fontSize: 14, color: 'text.secondary' }}>
            {t`Paste a Mortar link, a ${{ source: sourceLabel('nexus') }} collection, an ${{ manager: R2MODMAN }} or ${{ otherManager: GALE }} code, a ${{ otherSource: sourceLabel('patreon') }} post, an ${{ thirdSource: sourceLabel('itch') }} page or (Premium) a ${{ source: sourceLabel('nexus') }} mod link.`}
          </Typography>
          <Box
            component="form"
            onSubmit={(e) => {
              e.preventDefault()
              if (canPreview) {
                flow.previewLink(flow.text).catch(reportUnexpected)
              }
            }}
            sx={{ display: 'flex', alignItems: 'flex-start', gap: space.gap }}
          >
            <TextField
              size="small"
              fullWidth={true}
              autoFocus={true}
              placeholder={t`Link or code`}
              value={flow.text}
              onChange={(e) => flow.setText(e.target.value)}
              error={flow.error !== ''}
              helperText={flow.error || undefined}
              slotProps={{
                htmlInput: { 'aria-label': t`Link or code`, spellCheck: false },
                formHelperText: { role: 'alert', sx: { mx: 0, fontSize: 13 } },
                input: {
                  endAdornment: (
                    <InputAdornment position="end">
                      <Tooltip title={t`Paste`} describeChild={true}>
                        <span>
                          <IconButton
                            edge="end"
                            size="small"
                            aria-label={t`Paste`}
                            disabled={flow.busy}
                            onClick={() => {
                              flow.paste().catch(reportUnexpected)
                            }}
                          >
                            <ClipboardPaste size={16} />
                          </IconButton>
                        </span>
                      </Tooltip>
                    </InputAdornment>
                  ),
                },
              }}
              sx={{ userSelect: 'text' }}
            />
            <Button
              type="submit"
              variant="contained"
              disabled={!canPreview}
              sx={{ flexShrink: 0, whiteSpace: 'nowrap', height: space.control, minWidth: 104 }}
            >
              {flow.busy ? t`Reading…` : t`Preview`}
            </Button>
          </Box>
        </>
      )}
      {handoff ? (
        <Typography role="status" sx={{ fontSize: 14, color: 'text.secondary' }}>
          {handoff.kind === 'patreon'
            ? t`Opened the ${{ source: sourceLabel('patreon') }} post. Save the file from it; Mortar offers it from your Downloads folder.`
            : t`Opened the ${{ source: sourceLabel('itch') }} page. Save the file from it; Mortar offers it from your Downloads folder.`}
        </Typography>
      ) : null}
      {flow.tab === 'file' && flow.error ? (
        <Typography role="alert" sx={{ fontSize: 14, color: 'error.light' }}>
          {flow.error}
        </Typography>
      ) : null}
    </Box>
  )
}
