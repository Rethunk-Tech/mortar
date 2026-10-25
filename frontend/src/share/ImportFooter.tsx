import { useLingui } from '@lingui/react/macro'
import { Box, Button, FormControlLabel, Radio, RadioGroup, Typography } from '@mui/material'
import { ChevronRight } from 'lucide-react'
import { useState } from 'react'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { LockedReason } from '../mods/LockedReason.tsx'
import { useProfileLocked } from '../mods/useLocked.ts'
import { DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { Fold } from '../shell/Fold.tsx'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { Problems, StatusBar } from './ImportProblems.tsx'
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
            fontSize: 'base.fontSize',
          }}
        >
          <Box
            sx={{ flex: 1 }}
          >{t`Sign in to ${{ source: sourceLabel('nexus') }} to download these mods.`}</Box>
          <Button size="small" variant="outlined" color="inherit" onClick={flow.signIn}>
            {t`Open ${{ source: sourceLabel('nexus') }} settings`}
          </Button>
        </Box>
      ) : null}
      <StatusBar preview={preview} summary={summary} needsSignIn={needsSignIn} />
      <InstallMode flow={flow} preview={preview} />
      {flow.error ? (
        <Typography
          role="alert"
          sx={{ px: space.gap, pb: space.gap, fontSize: 'medium.fontSize', color: 'error.light' }}
        >
          {flow.error}
        </Typography>
      ) : null}
      <DialogFooter>
        {targetName ? (
          <LockedReason locked={targetLocked}>
            <Button
              variant="text"
              disabled={!canRun || targetLocked}
              onClick={() => {
                flow.run(true).catch(reportUnexpected)
              }}
            >
              {t`Add to ${{ name: targetName }}`}
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
            >
              {t`Replace ${{ name: targetName }}`}
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
        >
          {runLabel}
        </Button>
      </DialogFooter>
      <ConfirmationDialog
        open={askReplace}
        tone="danger"
        title={t`Replace ${{ name: targetName }}?`}
        confirmLabel={t`Replace ${{ name: targetName }}`}
        onCancel={() => setAskReplace(false)}
        onConfirm={() => {
          setAskReplace(false)
          flow.runReplace().catch(reportUnexpected)
        }}
      >
        {(preview.replace.remove ?? []).length > 0 ? (
          <Fold
            title={t`These will be removed (${{ count: (preview.replace.remove ?? []).length }})`}
          >
            <Typography
              tabIndex={0}
              sx={{ fontSize: 'base.fontSize', maxHeight: 120, overflowY: 'auto', mt: 0.5 }}
            >
              {(preview.replace.remove ?? []).join(', ')}
            </Typography>
          </Fold>
        ) : (
          <Typography sx={{ fontSize: 'base.fontSize' }}>{t`Nothing will be removed.`}</Typography>
        )}
        {(preview.replace.keepLocal ?? []).length > 0 ? (
          <Fold
            title={t`Local-only mods kept (${{ count: (preview.replace.keepLocal ?? []).length }})`}
          >
            <Typography
              tabIndex={0}
              sx={{ fontSize: 'base.fontSize', maxHeight: 120, overflowY: 'auto', mt: 0.5 }}
            >
              {(preview.replace.keepLocal ?? []).join(', ')}
            </Typography>
          </Fold>
        ) : null}
      </ConfirmationDialog>
    </Box>
  )
}

// Whether a share's missing dependencies are filled in or it installs exactly as shared: a modpack missing a mod on
// purpose (a lobby whose host left one out) must not gain it. Shown only when the share could bring dependencies.
function InstallMode({ flow, preview }: { flow: ImportFlow; preview: ShownPreview }) {
  const { t } = useLingui()
  if (!preview.mods.some((m) => m.site === 'thunderstore' || m.state === 'dependency')) {
    return null
  }
  return (
    <RadioGroup
      row={true}
      aria-label={t`How to install`}
      value={flow.asIs ? 'as-is' : 'fill'}
      onChange={(e) => flow.setAsIs(e.target.value === 'as-is')}
      sx={{ px: space.gap }}
    >
      <FormControlLabel
        value="fill"
        control={<Radio size="small" />}
        label={t`Fill in missing dependencies`}
      />
      <FormControlLabel
        value="as-is"
        control={<Radio size="small" />}
        label={t`As shared: same versions, no added dependencies (mods may be outdated or broken)`}
      />
    </RadioGroup>
  )
}
