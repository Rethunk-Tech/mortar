import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { memo, useRef, useState } from 'react'
import { MoveDataFolderPreview } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/service.ts'
import { useNav } from '../../nav/store.ts'
import { space } from '../../theme/density.ts'
import { type InlineError, reportError } from '../../toasts/report.ts'
import { PrefKeys } from '../PrefRow.tsx'
import { SettingsSection } from '../SettingsSection.tsx'
import { CacheClearDialog } from './DataMods.tsx'
import { MoveDialog } from './DataMove.tsx'
import { type MoveState, moveDataFolder } from './DataMoveRun.ts'
import { StoreCheckRow } from './DataStoreCheck.tsx'
import { CleanupDialog } from './DataStoreReport.tsx'
import { Location, UsageRows } from './DataUsage.tsx'
import { useDataUsage, useShownOnce } from './DataUsageLoad.ts'

function useMove() {
  const { t } = useLingui()
  const [move, setMove] = useState<MoveState | null>(null)
  const [moveError, setMoveError] = useState<InlineError | null>(null)
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
        setMoveError(null)
        setMove({ dest, estimate })
      })
      .catch((err: unknown) => {
        reportError(t`Could not inspect the destination folder`, () => prepare(dest))(err)
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

// Memoised: the disk use arriving re-renders the usage rows above, not these.
const Upkeep = memo(function UpkeepView() {
  const { t } = useLingui()
  return (
    <>
      <SettingsSection title={t`Integrity`}>
        <StoreCheckRow />
      </SettingsSection>
      <SettingsSection title={t`Retention`}>
        <PrefKeys keys={['storeRetentionDays', 'trashRetentionDays', 'historyEventsKept']} />
      </SettingsSection>
    </>
  )
})

export function Storage() {
  const { t } = useLingui()
  const openProfiles = useNav((s) => s.openProfiles)
  const sentinel = useRef<HTMLSpanElement>(null)
  const { usage, restart, error } = useDataUsage(useShownOnce(sentinel))
  const move = useMove()
  const [clearing, setClearing] = useState(false)
  const [cleaning, setCleaning] = useState(false)
  return (
    <>
      <span ref={sentinel} />
      <Location usage={usage} onPicked={move.prepare} />
      {error ? (
        <Box
          role="alert"
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: space.gap,
            px: space.pad,
            py: space.pad,
            fontSize: 15,
          }}
        >
          {t`Could not measure disk use`}
          <Button variant="outlined" onClick={restart}>
            {t`Retry`}
          </Button>
        </Box>
      ) : (
        <UsageRows
          usage={usage}
          onCleanUp={() => setCleaning(true)}
          onClearCache={() => setClearing(true)}
          onDeletedProfiles={openProfiles}
        />
      )}
      <Upkeep />
      {move.dialog}
      <CleanupDialog open={cleaning} onClose={() => setCleaning(false)} onChanged={restart} />
      <CacheClearDialog open={clearing} onClose={() => setClearing(false)} onCleared={restart} />
    </>
  )
}
