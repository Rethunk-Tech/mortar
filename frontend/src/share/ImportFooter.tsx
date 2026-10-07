import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { ChevronRight } from 'lucide-react'
import { useState } from 'react'
import { LockedReason } from '../mods/LockedReason.tsx'
import { useProfileLocked } from '../mods/useLocked.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { Fold } from '../shell/Fold.tsx'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { space } from '../theme/density.ts'
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
  // A share sent from a nearby Mortar shows under the file tab, but nobody picked a file.
  const fromNearby = useImportDialog((s) => s.request?.tab === 'data')
  let runLabel = flow.tab === 'file' ? t`New profile from file` : t`New profile from link`
  if (flow.external || fromNearby) {
    runLabel = t`New profile`
  }
  const [askReplace, setAskReplace] = useState(false)
  const closeDialog = useImportDialog((s) => s.close)
  const targetId = useImportDialog((s) => s.request?.profileId ?? '')
  const targetLocked = useProfileLocked(targetId)
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', flexShrink: 0 }}>
      <Problems preview={preview} excluded={flow.excluded} onLeaveOut={flow.toggle} />
      {needsSignIn ? (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: space.gap,
            m: '8px 8px 0',
            p: space.gap,
            bgcolor: calloutFill('info'),
            border: '1px solid',
            borderColor: calloutLine('info'),
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
      <StatusBar preview={preview} summary={summary} needsSignIn={needsSignIn} />
      {flow.error ? (
        <Typography
          role="alert"
          sx={{ px: space.gap, pb: space.gap, fontSize: 14, color: 'error.light' }}
        >
          {flow.error}
        </Typography>
      ) : null}
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: space.gap,
          px: space.gap,
          pb: space.gap,
        }}
      >
        <Button
          variant="text"
          color="inherit"
          onClick={closeDialog}
          disabled={flow.busy}
          sx={{ height: 40 }}
        >
          {t`Close`}
        </Button>
        <Button
          variant="text"
          color="inherit"
          onClick={flow.reset}
          disabled={flow.busy}
          sx={{ height: 40 }}
        >
          {t`Reset`}
        </Button>
        <Box sx={{ flex: 1 }} />
        {targetName ? (
          <LockedReason locked={targetLocked}>
            <Button
              variant="text"
              disabled={!canRun || targetLocked}
              onClick={() => {
                flow.run(true).catch(reportUnexpected)
              }}
              sx={{ height: 40 }}
            >
              {t`Add to ${targetName}`}
            </Button>
          </LockedReason>
        ) : null}
        {targetName ? (
          <LockedReason locked={targetLocked}>
            <Button
              variant="text"
              color="error"
              disabled={!canRun || targetLocked}
              onClick={() => setAskReplace(true)}
              sx={{ height: 40 }}
            >
              {t`Replace ${targetName}`}
            </Button>
          </LockedReason>
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
      <ConfirmDialog
        open={askReplace}
        color="error"
        title={t`Replace ${{ name: targetName }}?`}
        confirmLabel={t`Replace ${targetName}`}
        onCancel={() => setAskReplace(false)}
        onConfirm={() => {
          setAskReplace(false)
          flow.runReplace().catch(reportUnexpected)
        }}
      >
        {(preview.replace?.remove ?? []).length > 0 ? (
          <Fold title={t`These will be removed (${(preview.replace?.remove ?? []).length})`}>
            <Typography
              tabIndex={0}
              sx={{ fontSize: 13, maxHeight: 120, overflowY: 'auto', mt: 0.5 }}
            >
              {(preview.replace?.remove ?? []).join(', ')}
            </Typography>
          </Fold>
        ) : (
          <Typography sx={{ fontSize: 13 }}>{t`Nothing will be removed.`}</Typography>
        )}
        {(preview.replace?.keepLocal ?? []).length > 0 ? (
          <Fold title={t`Local-only mods kept (${(preview.replace?.keepLocal ?? []).length})`}>
            <Typography
              tabIndex={0}
              sx={{ fontSize: 13, maxHeight: 120, overflowY: 'auto', mt: 0.5 }}
            >
              {(preview.replace?.keepLocal ?? []).join(', ')}
            </Typography>
          </Fold>
        ) : null}
      </ConfirmDialog>
    </Box>
  )
}
