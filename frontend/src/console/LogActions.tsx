import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { Download, LifeBuoy } from 'lucide-react'
import { RunLog } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { SaveFile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import { Log } from '../../bindings/github.com/Rethunk-AI/mortar/internal/support/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { firstError, formatAll } from './filter.ts'
import { useShownEntries, useVisible } from './logHooks.ts'
import { logFileName, saveLogText } from './save.ts'
import { useConsole } from './store.ts'

const actionSx = {
  height: 32,
  whiteSpace: 'nowrap',
  borderColor: 'rgba(255,255,255,0.22)',
  color: '#ffffff',
} as const

export function LogActions() {
  const { t } = useLingui()
  const rows = useVisible()
  const entries = useShownEntries()
  const shown = useConsole((s) => s.shown)
  const viewingRun = useConsole((s) => s.viewingRun)
  const cleared = useConsole((s) => s.cleared)
  const profileName = useProfiles(
    (s) => s.profiles.find((p) => p.id === shown.profile)?.name ?? shown.profile,
  )
  const { clear, jumpTo, setHelping } = useConsole.getState()
  const firstErr = firstError(rows)
  const canSave = entries.length > 0 || cleared > 0
  return (
    <Box sx={{ display: 'flex', gap: 1 }}>
      <Button
        variant="outlined"
        color="inherit"
        disabled={firstErr < 0}
        onClick={() => jumpTo(firstErr)}
        sx={actionSx}
      >
        {t`Jump to first error`}
      </Button>
      <Button
        variant="outlined"
        color="inherit"
        disabled={rows.length === 0}
        onClick={clear}
        sx={actionSx}
      >
        {t`Clear`}
      </Button>
      <Button
        variant="outlined"
        color="inherit"
        disabled={rows.length === 0}
        onClick={() => {
          Clipboard.SetText(formatAll(rows)).then(
            () => useToasts.getState().push({ kind: 'success', title: t`Log copied` }),
            reportUnexpected,
          )
        }}
        sx={actionSx}
      >
        {t`Copy`}
      </Button>
      <Button
        variant="outlined"
        color="inherit"
        disabled={!canSave}
        startIcon={<Download size={16} />}
        onClick={() => {
          const rawLog = viewingRun
            ? RunLog(shown.game, shown.profile, viewingRun)
            : Log(shown.game, shown.profile)
          rawLog
            .then((raw) => {
              const text = saveLogText(raw ?? '', entries)
              if (text === '') {
                return
              }
              return SaveFile(t`Save log`, logFileName(profileName, new Date()), text)
            })
            .catch(reportUnexpected)
        }}
        sx={actionSx}
      >
        {t`Save log…`}
      </Button>
      <Button
        variant="outlined"
        color="inherit"
        startIcon={<LifeBuoy size={16} />}
        onClick={() => setHelping(true)}
        sx={actionSx}
      >
        {t`Get help`}
      </Button>
    </Box>
  )
}
