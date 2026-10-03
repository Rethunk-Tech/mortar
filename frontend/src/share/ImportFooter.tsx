import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { ChevronRight } from 'lucide-react'
import { useState } from 'react'
import { useLaunch } from '../launch/store.ts'
import { isLocked } from '../mods/locked.ts'
import { paper } from '../mods/paper.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { Problems, StatusBar } from './ImportPreview.tsx'
import type { ShownPreview, Summary } from './logic.ts'
import { useImportDialog } from './store.ts'
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
  let runLabel = flow.tab === 'file' ? t`New profile from file` : t`New profile from link`
  if (flow.external) {
    runLabel = t`New profile`
  }
  const [askReplace, setAskReplace] = useState(false)
  const targetId = useImportDialog((s) => s.request?.profileId ?? '')
  const targetLocked = isLocked(
    useLaunch((s) => s.status),
    targetId,
  )
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
          gridTemplateColumns: targetName ? '1fr 1fr 1fr 2fr' : '1fr 2fr',
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
          <DisabledReason title={t`Stop the game to change mods.`} disabled={targetLocked}>
            <Button
              variant="outlined"
              disabled={!canRun || targetLocked}
              onClick={() => {
                flow.run(true).catch(reportUnexpected)
              }}
              sx={{ height: 40 }}
            >
              {t`Add to ${targetName}`}
            </Button>
          </DisabledReason>
        ) : null}
        {targetName ? (
          <DisabledReason title={t`Stop the game to change mods.`} disabled={targetLocked}>
            <Button
              variant="outlined"
              disabled={!canRun || targetLocked}
              onClick={() => setAskReplace(true)}
              sx={{ height: 40 }}
            >
              {t`Replace ${targetName}`}
            </Button>
          </DisabledReason>
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
          {runLabel}
        </Button>
      </Box>
      <Dialog
        open={askReplace}
        onClose={() => setAskReplace(false)}
        transitionDuration={0}
        slotProps={{ paper }}
      >
        <DialogTitle>{t`Replace ${targetName}?`}</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 1, minWidth: 360 }}>
          {(preview.replace?.remove ?? []).length > 0 ? (
            <Box component="details">
              <Typography component="summary" sx={{ fontSize: 13, cursor: 'pointer' }}>
                {t`These will be removed (${(preview.replace?.remove ?? []).length})`}
              </Typography>
              <Typography sx={{ fontSize: 13, maxHeight: 120, overflowY: 'auto', mt: 0.5 }}>
                {(preview.replace?.remove ?? []).join(', ')}
              </Typography>
            </Box>
          ) : (
            <Typography sx={{ fontSize: 13 }}>{t`Nothing will be removed.`}</Typography>
          )}
          {(preview.replace?.keepLocal ?? []).length > 0 ? (
            <Box component="details">
              <Typography component="summary" sx={{ fontSize: 13, cursor: 'pointer' }}>
                {t`Local-only mods kept (${(preview.replace?.keepLocal ?? []).length})`}
              </Typography>
              <Typography sx={{ fontSize: 13, maxHeight: 120, overflowY: 'auto', mt: 0.5 }}>
                {(preview.replace?.keepLocal ?? []).join(', ')}
              </Typography>
            </Box>
          ) : null}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAskReplace(false)}>{t`Cancel`}</Button>
          <Button
            color="error"
            onClick={() => {
              setAskReplace(false)
              flow.runReplace().catch(reportUnexpected)
            }}
          >
            {t`Replace ${targetName}`}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
