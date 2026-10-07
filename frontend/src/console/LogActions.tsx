import { useLingui } from '@lingui/react/macro'
import { Box, Menu } from '@mui/material'
import {
  ChevronDown,
  ChevronUp,
  CircleAlert,
  Copy,
  Download,
  Ellipsis,
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
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { firstError, formatAll } from './filter.ts'
import { useShownEntries, useVisible } from './logHooks.ts'
import { SearchRunsDialog } from './SearchRunsDialog.tsx'
import { logFileName, saveLogText } from './save.ts'
import { useConsole } from './store.ts'
import { useConsoleEmpty } from './useConsoleEmpty.ts'

// Share log sits beside the menu for a loader that uploads to smapi.io or has a paste site; Save log lives in the menu.
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
  const [menu, setMenu] = useState<HTMLElement | null>(null)
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
  const pick = (run: () => void) => () => {
    setMenu(null)
    run()
  }
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
        label={t`Log actions`}
        icon={<Ellipsis size={16} />}
        menu={true}
        aria-haspopup="menu"
        aria-expanded={menu !== null}
        onClick={(e) => setMenu(e.currentTarget)}
      />
      <ShareLog />
      <Menu anchorEl={menu} open={menu !== null} onClose={() => setMenu(null)}>
        <MenuAction
          disabled={rows.length === 0}
          icon={<Copy size={16} />}
          label={t`Copy log`}
          onClick={pick(() => {
            copyText(formatAll(rows), t`Log copied`)
          })}
        />
        <MenuAction
          disabled={!canSave}
          icon={<Download size={16} />}
          label={t`Save log…`}
          onClick={pick(saveLog)}
        />
        <MenuAction
          icon={<FileSearch size={16} />}
          label={t`Search all runs…`}
          onClick={pick(() => setSearching(true))}
        />
        <MenuAction
          disabled={rows.length === 0}
          icon={<Eraser size={16} />}
          label={t`Clear`}
          onClick={pick(clear)}
        />
      </Menu>
      <SearchRunsDialog open={searching} onClose={() => setSearching(false)} />
    </PageActions>
  )
}
