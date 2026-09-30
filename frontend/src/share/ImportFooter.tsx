import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { ChevronRight } from 'lucide-react'
import { reportUnexpected } from '../toasts/report.ts'
import { Problems, StatusBar } from './ImportPreview.tsx'
import type { ShownPreview, Summary } from './logic.ts'
import type { ImportFlow } from './useImportFlow.ts'

// The preview's problems, status bar and actions. Nothing downloads until one of the two import buttons is pressed.
export function ImportFooter({
  flow,
  preview,
  summary,
  targetName,
  signedIn,
}: {
  flow: ImportFlow
  preview: ShownPreview
  summary: Summary
  targetName: string
  signedIn: boolean
}) {
  const { t } = useLingui()
  const needsSignIn = summary.fromNexus > 0 && !signedIn
  const canRun = !(flow.busy || needsSignIn)
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', flexShrink: 0 }}>
      <Problems preview={preview} excluded={flow.excluded} onLeaveOut={flow.toggle} />
      {needsSignIn ? (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 1.5,
            m: '8px 8px 0',
            p: 1.25,
            bgcolor: 'rgba(43,139,218,0.16)',
            border: '1px solid rgba(43,139,218,0.5)',
            borderRadius: '4px',
            fontSize: 13,
          }}
        >
          <Box sx={{ flex: 1 }}>{t`Sign in to Nexus Mods to download these mods.`}</Box>
          <Button size="small" variant="outlined" color="inherit" onClick={flow.signIn}>
            {t`Open Nexus settings`}
          </Button>
        </Box>
      ) : null}
      <StatusBar preview={preview} summary={summary} />
      {flow.error ? (
        <Typography role="alert" sx={{ px: 1, pb: 1, fontSize: 14, color: 'error.light' }}>
          {flow.error}
        </Typography>
      ) : null}
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: targetName ? '1fr 1fr 2fr' : '1fr 2fr',
          gap: 1,
          px: 1,
          pb: 1,
        }}
      >
        <Button
          variant="outlined"
          color="inherit"
          onClick={flow.reset}
          disabled={flow.busy}
          sx={{ height: 40 }}
        >
          {t`Reset`}
        </Button>
        {targetName ? (
          <Button
            variant="outlined"
            disabled={!canRun}
            onClick={() => {
              flow.run(true).catch(reportUnexpected)
            }}
            sx={{ height: 40 }}
          >
            {t`Add to ${targetName}`}
          </Button>
        ) : null}
        <Button
          variant="contained"
          endIcon={<ChevronRight size={14} />}
          disabled={!canRun}
          onClick={() => {
            flow.run(false).catch(reportUnexpected)
          }}
          sx={{ height: 40 }}
        >
          {flow.tab === 'file' ? t`New profile from file` : t`New profile from link`}
        </Button>
      </Box>
    </Box>
  )
}
