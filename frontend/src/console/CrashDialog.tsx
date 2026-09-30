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
import { useTab } from '../game/tab.ts'
import { useLaunch } from '../launch/store.ts'
import { paper } from '../mods/paper.ts'
import { useConsole } from './store.ts'

export function CrashDialog() {
  const { t } = useLingui()
  const crash = useLaunch((s) => s.crash)
  const dismiss = useLaunch((s) => s.dismissCrash)
  if (!crash) {
    return null
  }
  return (
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
          crash.mods.map((row) => (
            <Box key={row.mod} sx={{ fontSize: 14, lineHeight: 1.45 }}>
              <Box sx={{ fontWeight: 700 }}>{t`${row.mod} · ${row.count} errors`}</Box>
              <Box sx={{ color: 'text.secondary', mt: 0.25 }}>{row.first}</Box>
            </Box>
          ))
        )}
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2.5 }}>
        <Button onClick={dismiss} sx={{ whiteSpace: 'nowrap' }}>
          {t`Dismiss`}
        </Button>
        <Button
          variant="outlined"
          startIcon={<LifeBuoy size={16} />}
          onClick={() => {
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
  )
}
