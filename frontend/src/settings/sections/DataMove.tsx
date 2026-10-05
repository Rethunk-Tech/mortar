import { useLingui } from '@lingui/react/macro'
import { Box, Button, LinearProgress } from '@mui/material'
import { FolderInput } from 'lucide-react'
import { PickFolder } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { formatBytes } from '../../i18n/bytes.ts'
import { useGameBusy } from '../../launch/store.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { ErrorRetry } from '../../shell/ErrorRetry.tsx'
import { type InlineError, reportError } from '../../toasts/report.ts'
import type { MoveState } from './DataMoveRun.ts'
import { mono, nowrap } from './dataStyles.ts'

const PERCENT = 100

export function MoveDataButton({
  onPicked,
  disabledReason: portableReason = '',
}: {
  onPicked: (dest: string) => void
  disabledReason?: string
}) {
  const { t } = useLingui()
  const busy = useGameBusy()
  const disabledReason =
    portableReason || (busy ? t`Stop the game before moving the data folder.` : '')
  return (
    <DisabledReason title={disabledReason} disabled={disabledReason !== ''}>
      <Button
        disabled={disabledReason !== ''}
        variant="outlined"
        startIcon={<FolderInput size={16} />}
        onClick={() => {
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
  error: InlineError | null
  onClose: () => void
  onMove: () => void
}) {
  const { t } = useLingui()
  const short = move !== null && move.estimate.freeBytes < move.estimate.bytes
  return (
    <ConfirmDialog
      open={move !== null}
      title={t`Move data folder`}
      confirmLabel={t`Move`}
      busy={moving}
      confirmDisabled={move === null || short}
      maxWidth={420}
      onCancel={onClose}
      onConfirm={onMove}
    >
      {move ? (
        <>
          <Box sx={{ ...mono, overflowWrap: 'anywhere' }} title={move.dest}>
            {move.dest}
          </Box>
          <Box>{t`Data to copy: ${formatBytes(move.estimate.bytes)}`}</Box>
          <Box>{t`Free space at destination: ${formatBytes(move.estimate.freeBytes)}`}</Box>
          {short ? (
            <Box role="alert" sx={{ color: 'error.main' }}>
              {t`There is not enough free space at the destination.`}
            </Box>
          ) : null}
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
          {error ? <ErrorRetry error={error} onRetry={onMove} /> : null}
        </>
      ) : null}
    </ConfirmDialog>
  )
}
