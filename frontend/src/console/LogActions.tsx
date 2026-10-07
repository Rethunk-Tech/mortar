import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import {
  ChevronDown,
  ChevronUp,
  CircleAlert,
  Copy,
  Download,
  Eraser,
  FileSearch,
  LifeBuoy,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { Level } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import {
  RunLog,
  Runs,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { SaveFile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { Log } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'
import { PageActions } from '../game/PageActions.tsx'
import { useProfileLoader, useProfiles } from '../profiles/store.ts'
import { copyText } from '../share/copyText.ts'
import { IconAction } from '../shell/IconAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { firstError, formatAll } from './filter.ts'
import { useShownEntries, useVisible } from './logHooks.ts'
import { SearchRunsDialog } from './SearchRunsDialog.tsx'
import { logFileName, saveLogText } from './save.ts'
import { useConsole } from './store.ts'
import { useConsoleEmpty } from './useConsoleEmpty.ts'

// Share log sits after the log actions for a loader that uploads to smapi.io or has a paste site.
function ShareLog() {
  const { t } = useLingui()
  const loader = useProfileLoader()
  if (loader?.share !== true && !loader?.paste) {
    return null
  }
  return (
    <IconAction
      label={t`Share log…`}
      icon={<LifeBuoy size={16} />}
      onClick={() => useConsole.getState().setHelping(true)}
    />
  )
}

export function LogActions({ game }: { game: string }) {
  const { t } = useLingui()
  const empty = useConsoleEmpty(game)
  const rows = useVisible()
  const entries = useShownEntries()
  const shown = useConsole((s) => s.shown)
  const viewingRun = useConsole((s) => s.viewingRun)
  const cleared = useConsole((s) => s.cleared)
  const profileName = useProfiles(
    (s) => s.profiles.find((p) => p.id === shown.profile)?.name ?? shown.profile,
  )
  const { clear, jumpTo } = useConsole.getState()
  const loaderName = useProfileLoader()?.name ?? ''
  const [searching, setSearching] = useState(false)
  const [viewedStart, setViewedStart] = useState<Date | null>(null)
  useEffect(() => {
    if (!viewingRun) {
      setViewedStart(null)
      return
    }
    Runs(shown.game, shown.profile)
      .then((runs) => {
        const run = (runs ?? []).find((item) => item.id === viewingRun)
        setViewedStart(run?.started ? new Date(run.started) : null)
      })
      .catch(() => setViewedStart(null))
  }, [shown.game, shown.profile, viewingRun])
  const firstErr = firstError(rows)
  const errorRows = rows.flatMap((row, index) =>
    row.level === Level.Error && !row.cont ? [index] : [],
  )
  const firstErrorPosition = errorRows.indexOf(firstErr)
  const previousErr = firstErrorPosition > 0 ? (errorRows[firstErrorPosition - 1] ?? -1) : -1
  const nextErr =
    firstErrorPosition >= 0 && firstErrorPosition < errorRows.length - 1
      ? (errorRows[firstErrorPosition + 1] ?? -1)
      : -1
  const canSave = entries.length > 0 || cleared > 0
  const saveLog = () => {
    const rawLog = viewingRun
      ? RunLog(shown.game, shown.profile, viewingRun)
      : Log(shown.game, shown.profile)
    rawLog
      .then((raw) => {
        const text = saveLogText(raw ?? '', entries)
        if (text === '') {
          return
        }
        return SaveFile(
          t`Save log`,
          logFileName(loaderName, profileName, viewedStart ?? new Date()),
          text,
        )
      })
      .catch(reportUnexpected)
  }
  if (empty) {
    return null
  }
  return (
    <PageActions>
      <Box role="group" aria-label={t`Errors`} sx={{ display: 'flex', gap: '2px' }}>
        <IconAction
          label={t`Jump to first error`}
          icon={<CircleAlert size={16} />}
          disabled={firstErr < 0}
          onClick={() => jumpTo(firstErr)}
        />
        <IconAction
          label={t`Jump to previous error`}
          icon={<ChevronUp size={16} />}
          disabled={previousErr < 0}
          onClick={() => jumpTo(previousErr)}
        />
        <IconAction
          label={t`Jump to next error`}
          icon={<ChevronDown size={16} />}
          disabled={nextErr < 0}
          onClick={() => jumpTo(nextErr)}
        />
      </Box>
      <IconAction
        label={t`Copy log`}
        icon={<Copy size={16} />}
        disabled={rows.length === 0}
        onClick={() => copyText(formatAll(rows), t`Log copied`)}
      />
      <IconAction
        label={t`Save log…`}
        icon={<Download size={16} />}
        disabled={!canSave}
        onClick={saveLog}
      />
      <IconAction
        label={t`Search all runs…`}
        icon={<FileSearch size={16} />}
        onClick={() => setSearching(true)}
      />
      <IconAction
        label={t`Clear`}
        icon={<Eraser size={16} />}
        disabled={rows.length === 0}
        onClick={clear}
      />
      <ShareLog />
      <SearchRunsDialog open={searching} onClose={() => setSearching(false)} />
    </PageActions>
  )
}
