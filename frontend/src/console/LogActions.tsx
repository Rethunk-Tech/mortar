import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { CircleAlert, Copy, Download, Eraser, LifeBuoy } from 'lucide-react'
import { RunLog } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { SaveFile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import { Log } from '../../bindings/github.com/Rethunk-AI/mortar/internal/support/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { firstError, formatAll } from './filter.ts'
import { IconAction } from './IconAction.tsx'
import { useShownEntries, useVisible } from './logHooks.ts'
import { logFileName, saveLogText } from './save.ts'
import { useConsole } from './store.ts'

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
    <Box sx={{ display: 'flex', gap: 0.75 }}>
      <IconAction
        label={t`Jump to first error`}
        icon={<CircleAlert size={16} />}
        disabled={firstErr < 0}
        onClick={() => jumpTo(firstErr)}
      />
      <IconAction
        label={t`Clear`}
        icon={<Eraser size={16} />}
        disabled={rows.length === 0}
        onClick={clear}
      />
      <IconAction
        label={t`Copy`}
        icon={<Copy size={16} />}
        disabled={rows.length === 0}
        onClick={() => {
          Clipboard.SetText(formatAll(rows)).then(
            () => useToasts.getState().push({ kind: 'success', title: t`Log copied` }),
            reportUnexpected,
          )
        }}
      />
      <IconAction
        label={t`Save log…`}
        icon={<Download size={16} />}
        disabled={!canSave}
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
      />
      <IconAction
        label={t`Get help`}
        icon={<LifeBuoy size={16} />}
        onClick={() => setHelping(true)}
      />
    </Box>
  )
}
