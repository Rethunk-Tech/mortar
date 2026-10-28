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
import { memo, useCallback, useRef, useState } from 'react'
import { Level } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { RunLog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { SaveFile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { Log } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'
import { PageActions } from '../game/PageActions.tsx'
import { useProfileLoader, useProfiles } from '../profiles/store.ts'
import { copyText } from '../share/copyText.ts'
import { IconAction } from '../shell/IconAction.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { firstError, formatAll } from './filter.ts'
import { useLineText } from './lineText.ts'
import { listRuns } from './listRuns.ts'
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

// The buttons take only what they show, so a log, run or mod list arriving redraws them only when one turns on or off.
const Buttons = memo(function ButtonsView({
  firstErr,
  previousErr,
  nextErr,
  hasRows,
  canSave,
  onCopy,
  onSave,
}: {
  firstErr: number
  previousErr: number
  nextErr: number
  hasRows: boolean
  canSave: boolean
  onCopy: () => void
  onSave: () => void
}) {
  const { t } = useLingui()
  const [searching, setSearching] = useState(false)
  const { clear, jumpTo } = useConsole.getState()
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
        disabled={!hasRows}
        onClick={onCopy}
      />
      <IconAction
        label={t`Save log…`}
        icon={<Download size={16} />}
        disabled={!canSave}
        onClick={onSave}
      />
      <IconAction
        label={t`Search all runs…`}
        icon={<FileSearch size={16} />}
        onClick={() => setSearching(true)}
      />
      <IconAction
        label={t`Clear`}
        icon={<Eraser size={16} />}
        disabled={!hasRows}
        onClick={clear}
      />
      <ShareLog />
      <SearchRunsDialog open={searching} onClose={() => setSearching(false)} />
    </PageActions>
  )
})

export function LogActions({ game }: { game: string }) {
  const { t } = useLingui()
  const empty = useConsoleEmpty(game)
  const rows = useVisible()
  const entries = useShownEntries()
  const shown = useConsole((s) => s.shown)
  const viewingRun = useConsole((s) => s.viewingRun)
  const jumpedTo = useConsole((s) => s.jump?.index ?? -1)
  const cleared = useConsole((s) => s.cleared)
  const profileName = useProfiles(
    (s) => s.profiles.find((p) => p.id === shown.profile)?.name ?? shown.profile,
  )
  const loaderName = useProfileLoader()?.name ?? ''
  const { data: viewedStart } = useLoaded<Date | null>(
    viewingRun
      ? () =>
          listRuns(shown.game, shown.profile).then(
            (runs) => {
              const run = runs.find((item) => item.id === viewingRun)
              return run?.started ? new Date(run.started) : null
            },
            () => null,
          )
      : null,
    [shown.game, shown.profile, viewingRun],
    null,
  )
  const firstErr = firstError(rows)
  const errorRows = rows.flatMap((row, index) =>
    row.level === Level.Error && !row.cont ? [index] : [],
  )
  // Previous and Next step from the row last jumped to; before any jump, Next is the first error.
  const position = errorRows.indexOf(jumpedTo)
  const previousErr = position > 0 ? (errorRows[position - 1] ?? -1) : -1
  const nextErr = errorRows[position + 1] ?? -1
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
  const lineText = useLineText()
  const latest = useRef({ rows, saveLog, lineText })
  latest.current = { rows, saveLog, lineText }
  const onCopy = useCallback(
    () => copyText(formatAll(latest.current.rows, latest.current.lineText), t`Log copied`),
    [t],
  )
  const onSave = useCallback(() => latest.current.saveLog(), [])
  if (empty) {
    return null
  }
  return (
    <Buttons
      firstErr={firstErr}
      previousErr={previousErr}
      nextErr={nextErr}
      hasRows={rows.length > 0}
      canSave={canSave}
      onCopy={onCopy}
      onSave={onSave}
    />
  )
}
