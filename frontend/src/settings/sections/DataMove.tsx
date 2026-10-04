import { useLingui } from '@lingui/react/macro'
import { Box, Button, LinearProgress } from '@mui/material'
import { FolderInput } from 'lucide-react'
import { PickFolder } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import { formatBytes } from '../../i18n/bytes.ts'
import { gameBusy } from '../../launch/busy.ts'
import { useLaunch } from '../../launch/store.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { reportError } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import type { MoveState } from './DataMoveRun.ts'
import { mono, nowrap } from './dataStyles.ts'

const PERCENT = 100

export function MoveDataButton({
  onPicked,
  disabledReason = '',
}: {
  onPicked: (dest: string) => void
  disabledReason?: string
}) {
  const { t } = useLingui()
  return (
    <DisabledReason title={disabledReason} disabled={disabledReason !== ''}>
      <Button
        disabled={disabledReason !== ''}
        variant="outlined"
        startIcon={<FolderInput size={16} />}
        onClick={() => {
          if (gameBusy(useLaunch.getState().status)) {
            useToasts.getState().push({
              kind: 'error',
              title: t`Stop the game before moving the data folder.`,
            })
            return
          }
          PickFolder(t`Move data folder…`)
            .then((dest) => (dest ? onPicked(dest) : undefined))
            .catch((err: unknown) => {
              reportError(t`Could not prepare the data folder move`)(err)
            })
        }}
        sx={{ ...nowrap, flexShrink: 0 }}
      >
        {t`Move…`}
      </Button>
    </DisabledReason>
  )
}

export function MoveDialog({
  move,
  moving,
  progress,
  error,
  onClose,
  onMove,
}: {
  move: MoveState | null
  moving: boolean
  progress: { files: number; totalFiles: number; bytes: number; totalBytes: number }
  error: string
  onClose: () => void
  onMove: () => void
}) {
  const { t } = useLingui()
  return (
    <ConfirmDialog
      open={move !== null}
      title={t`Move data folder`}
      confirmLabel={t`Move`}
      busy={moving}
      confirmDisabled={move === null}
      maxWidth={360}
      onCancel={onClose}
      onConfirm={onMove}
    >
      {move ? (
        <>
          <Box>{t`Data to copy: ${formatBytes(move.estimate.bytes)}`}</Box>
          <Box>{t`Free space at destination: ${formatBytes(move.estimate.freeBytes)}`}</Box>
          {moving ? (
            <>
              <LinearProgress
                variant="determinate"
                value={progress.totalBytes ? (progress.bytes / progress.totalBytes) * PERCENT : 0}
                aria-label={t`Moving data`}
              />
              <Box role="status" sx={{ ...mono }}>
                {t`${progress.files}/${progress.totalFiles} files · ${formatBytes(progress.bytes)} / ${formatBytes(progress.totalBytes)}`}
              </Box>
            </>
          ) : null}
          {error ? (
            <Box role="alert" sx={{ color: 'error.main' }}>
              {error}
            </Box>
          ) : null}
        </>
      ) : null}
    </ConfirmDialog>
  )
}
