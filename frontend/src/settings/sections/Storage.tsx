import { useLingui } from '@lingui/react/macro'
import { useState } from 'react'
import { MoveDataFolderPreview } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/service.ts'
import { useNav } from '../../nav/store.ts'
import { reportError } from '../../toasts/report.ts'
import { PrefKeys } from '../PrefRow.tsx'
import { SettingsSection } from '../SettingsSection.tsx'
import { CacheClearDialog } from './DataMods.tsx'
import { MoveDialog } from './DataMove.tsx'
import { type MoveState, moveDataFolder } from './DataMoveRun.ts'
import { CleanupDialog } from './DataStoreReport.tsx'
import { BackupsKept, Location, UsageRows } from './DataUsage.tsx'
import { useDataUsage } from './DataUsageLoad.ts'

function useMove() {
  const { t } = useLingui()
  const [move, setMove] = useState<MoveState | null>(null)
  const [moveError, setMoveError] = useState('')
  const [moving, setMoving] = useState(false)
  const [moveProgress, setMoveProgress] = useState({
    files: 0,
    totalFiles: 0,
    bytes: 0,
    totalBytes: 0,
  })
  const prepare = (dest: string) => {
    MoveDataFolderPreview(dest)
      .then((estimate) => {
        setMoveError('')
        setMove({ dest, estimate })
      })
      .catch((err: unknown) => {
        reportError(t`Could not inspect the destination folder`)(err)
      })
  }
  const run = () => {
    if (move) {
      moveDataFolder({
        move,
        setMove,
        setMoveError,
        setMoving,
        setMoveProgress,
        moveErrorText: t`Could not move the data folder.`,
      })
    }
  }
  const dialog = (
    <MoveDialog
      move={move}
      moving={moving}
      progress={moveProgress}
      error={moveError}
      onClose={() => setMove(null)}
      onMove={run}
    />
  )
  return { prepare, dialog }
}

export function Storage() {
  const { t } = useLingui()
  const openProfiles = useNav((s) => s.openProfiles)
  const { usage, bytes, restart } = useDataUsage()
  const move = useMove()
  const [clearing, setClearing] = useState(false)
  const [cleaning, setCleaning] = useState(false)
  return (
    <>
      <Location usage={usage} onPicked={move.prepare} />
      <UsageRows
        usage={usage}
        bytes={bytes}
        onCleanUp={() => setCleaning(true)}
        onClearCache={() => setClearing(true)}
        onDeletedProfiles={openProfiles}
      />
      <SettingsSection title={t`Retention`}>
        <PrefKeys keys={['storeRetentionDays', 'trashRetentionDays', 'historyEventsKept']} />
        <BackupsKept />
      </SettingsSection>
      {move.dialog}
      <CleanupDialog open={cleaning} onClose={() => setCleaning(false)} onChanged={restart} />
      <CacheClearDialog open={clearing} onClose={() => setClearing(false)} onCleared={restart} />
    </>
  )
}
