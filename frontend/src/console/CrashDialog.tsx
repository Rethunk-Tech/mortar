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
import { LifeBuoy, Terminal } from 'lucide-react'
import { useState } from 'react'
import { Start as StartBisect } from '../../bindings/github.com/Rethunk-AI/mortar/internal/bisect/service.ts'
import { useTab } from '../game/tab.ts'
import { useLaunch } from '../launch/store.ts'
import { paper } from '../mods/paper.ts'
import { useProfiles } from '../profiles/store.ts'
import { BisectDialog } from './BisectDialog.tsx'
import { useConsole } from './store.ts'

export function CrashDialog() {
  const { t } = useLingui()
  const crash = useLaunch((s) => s.crash)
  const dismiss = useLaunch((s) => s.dismissCrash)
  const [bisectJob, setBisectJob] = useState<{ id: string; game: string; profile: string } | null>(
    null,
  )
  const [bisectError, setBisectError] = useState<string | null>(null)
  if (!crash) {
    return bisectJob ? (
      <BisectDialog
        game={bisectJob.game}
        profile={bisectJob.profile}
        jobID={bisectJob.id}
        onClose={() => setBisectJob(null)}
      />
    ) : null
  }
  const canBisect = crash.mods === null || crash.mods.length === 0
  const startBisect = async () => {
    try {
      const id = await StartBisect(crash.game, crash.profile)
      setBisectJob({ id, game: crash.game, profile: crash.profile })
      setBisectError(null)
      dismiss()
    } catch (error) {
      setBisectError(error instanceof Error ? error.message : String(error))
    }
  }
  return (
    <>
      <Dialog
        open={true}
        onClose={dismiss}
        transitionDuration={0}
        slotProps={{ paper: { sx: { ...paper.sx, width: 520, maxWidth: 'calc(100% - 32px)' } } }}
      >
        <DialogTitle
          sx={{ fontSize: 22, fontWeight: 700 }}
        >{t`Stardew Valley closed with errors`}</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          {crash.mods === null || crash.mods.length === 0 ? (
            <Typography sx={{ fontSize: 14 }}>{t`The SMAPI log has errors.`}</Typography>
          ) : (
            crash.mods.map((row: { mod: string; count: number; first: string }) => (
              <Box key={row.mod} sx={{ fontSize: 14, lineHeight: 1.45 }}>
                <Box sx={{ fontWeight: 700 }}>{t`${row.mod} · ${row.count} errors`}</Box>
                <Box sx={{ color: 'text.secondary', mt: 0.25 }}>{row.first}</Box>
              </Box>
            ))
          )}
          {bisectError ? <Typography color="error">{bisectError}</Typography> : null}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5 }}>
          <Button onClick={dismiss} sx={{ whiteSpace: 'nowrap' }}>
            {t`Dismiss`}
          </Button>
          {canBisect ? (
            <Button onClick={startBisect} sx={{ whiteSpace: 'nowrap' }}>
              {t`Find the mod causing this`}
            </Button>
          ) : null}
          <Button
            variant="outlined"
            startIcon={<LifeBuoy size={16} />}
            onClick={() => {
              useProfiles.getState().open(crash.profile)
              useConsole.getState().viewRun(crash.game, crash.profile, crash.runId)
              useConsole.getState().setHelping(true)
              dismiss()
            }}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Get help`}
          </Button>
          <Button
            variant="contained"
            startIcon={<Terminal size={16} />}
            onClick={() => {
              useProfiles.getState().open(crash.profile)
              useTab.getState().setTab('console')
              useConsole.getState().viewRun(crash.game, crash.profile, crash.runId)
              dismiss()
            }}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Open Console`}
          </Button>
        </DialogActions>
      </Dialog>
      {bisectJob ? (
        <BisectDialog
          game={bisectJob.game}
          profile={bisectJob.profile}
          jobID={bisectJob.id}
          onClose={() => setBisectJob(null)}
        />
      ) : null}
    </>
  )
}
