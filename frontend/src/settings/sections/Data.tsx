import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { Trash2 } from 'lucide-react'
import { useState } from 'react'
import type { Preview } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/models.ts'
import {
  Cleanup,
  CleanupPreview,
  MoveDataFolderPreview,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/service.ts'
import type { ImportPreview } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import { useNav } from '../../nav/store.ts'
import { errorText, reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { DataDialogs } from './DataDialogs.tsx'
import { CacheClearDialog, DataByMod, DataSettingsFiles } from './DataMods.tsx'
import { type MoveState, moveDataFolder } from './DataMoveRun.ts'
import { DataPrefs } from './DataPrefs.tsx'
import { BackupsKept, DataFolderCard, UsageSummary } from './DataUsage.tsx'
import { useDataUsage } from './DataUsageLoad.ts'
import { nowrap } from './dataStyles.ts'

function useDataSection() {
  const { t } = useLingui()
  const openProfiles = useNav((s) => s.openProfiles)
  const { usage, bytes, restart, rev } = useDataUsage()
  const [preview, setPreview] = useState<Preview | null>(null)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [importPreview, setImportPreview] = useState<ImportPreview | null>(null)
  const [busy, setBusy] = useState(false)
  const [move, setMove] = useState<MoveState | null>(null)
  const [moveError, setMoveError] = useState('')
  const [moving, setMoving] = useState(false)
  const [moveProgress, setMoveProgress] = useState({
    files: 0,
    totalFiles: 0,
    bytes: 0,
    totalBytes: 0,
  })
  const [confirmClear, setConfirmClear] = useState(false)
  const openPreview = () => {
    setPreviewOpen(true)
    setPreview(null)
    setPreviewLoading(true)
    CleanupPreview()
      .then(setPreview)
      .catch(reportUnexpected)
      .finally(() => setPreviewLoading(false))
  }
  const runCleanup = () => {
    if (!preview) {
      return
    }
    setBusy(true)
    Cleanup(preview)
      .then(() => {
        setPreview(null)
        setPreviewOpen(false)
        restart()
      })
      .catch(reportUnexpected)
      .finally(() => setBusy(false))
  }
  const prepareMove = (dest: string) => {
    MoveDataFolderPreview(dest)
      .then((estimate) => {
        setMoveError('')
        setMove({ dest, estimate })
      })
      .catch((err: unknown) => {
        const body = errorText(err)
        useToasts.getState().push({
          kind: 'error',
          title: t`Couldn't inspect the destination folder`,
          ...(body ? { body } : {}),
        })
      })
  }
  const runMove = () => {
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
  return {
    openProfiles,
    usage,
    bytes,
    restart,
    rev,
    preview,
    previewOpen,
    previewLoading,
    importPreview,
    busy,
    move,
    setMove,
    moving,
    moveProgress,
    moveError,
    confirmClear,
    setConfirmClear,
    setPreview,
    setPreviewOpen,
    setImportPreview,
    openPreview,
    runCleanup,
    prepareMove,
    runMove,
  }
}

export function Data() {
  const { t } = useLingui()
  const d = useDataSection()
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <DataFolderCard usage={d.usage} onPicked={d.prepareMove} />
      <DataSettingsFiles onImport={d.setImportPreview} />
      <UsageSummary
        usage={d.usage}
        bytes={d.bytes}
        openProfiles={d.openProfiles}
        onClearCache={() => d.setConfirmClear(true)}
      />
      <DataByMod key={d.rev} onCleanup={d.openPreview} onChanged={d.restart} />
      <Button
        onClick={d.openPreview}
        startIcon={<Trash2 size={16} />}
        sx={{ alignSelf: 'flex-start', ...nowrap }}
      >
        {t`Clean up unused`}
      </Button>
      <Box sx={{ fontSize: 14, fontWeight: 600, pt: 1 }}>{t`Save backups`}</Box>
      <BackupsKept />
      <DataPrefs />
      <DataDialogs
        preview={d.preview}
        previewOpen={d.previewOpen}
        previewLoading={d.previewLoading}
        busy={d.busy}
        onClosePreview={() => {
          d.setPreview(null)
          d.setPreviewOpen(false)
        }}
        onCleanup={d.runCleanup}
        importPreview={d.importPreview}
        onCloseImport={() => d.setImportPreview(null)}
        move={d.move}
        moving={d.moving}
        moveProgress={d.moveProgress}
        moveError={d.moveError}
        onCloseMove={() => d.setMove(null)}
        onMove={d.runMove}
      />
      <CacheClearDialog
        open={d.confirmClear}
        onClose={() => d.setConfirmClear(false)}
        onCleared={d.restart}
      />
    </Box>
  )
}
