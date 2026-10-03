import { useLingui } from '@lingui/react/macro'
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material'
import type { Preview } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/models.ts'
import type { ImportPreview } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import { paper } from '../../mods/paper.ts'
import { ImportSettingsDialog } from './DataImport.tsx'
import { MoveDialog } from './DataMove.tsx'
import type { MoveState } from './DataMoveRun.ts'
import { Row } from './DataUi.tsx'
import { nowrap } from './dataStyles.ts'

export function DataDialogs({
  preview,
  busy,
  onClosePreview,
  onCleanup,
  importPreview,
  onCloseImport,
  move,
  moving,
  moveProgress,
  moveError,
  onCloseMove,
  onMove,
}: {
  preview: Preview | null
  busy: boolean
  onClosePreview: () => void
  onCleanup: () => void
  importPreview: ImportPreview | null
  onCloseImport: () => void
  move: MoveState | null
  moving: boolean
  moveProgress: { files: number; totalFiles: number; bytes: number; totalBytes: number }
  moveError: string
  onCloseMove: () => void
  onMove: () => void
}) {
  const { t } = useLingui()
  return (
    <>
      <Dialog
        open={preview !== null}
        onClose={onClosePreview}
        transitionDuration={0}
        slotProps={{ paper }}
      >
        <DialogTitle>{t`Clean up unused`}</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 1, minWidth: 360 }}>
          {(preview?.items ?? []).length === 0 ? (
            <Box sx={{ fontSize: 13 }}>{t`Nothing to remove.`}</Box>
          ) : (
            (preview?.items ?? []).map((it) => <Row key={it.rel} label={it.label} size={it.size} />)
          )}
          {(preview?.items ?? []).length > 0 ? (
            <Row label={t`Total`} size={preview?.total ?? 0} />
          ) : null}
        </DialogContent>
        <DialogActions>
          <Button onClick={onClosePreview} sx={nowrap}>
            {t`Cancel`}
          </Button>
          <Button
            onClick={onCleanup}
            disabled={busy || (preview?.items ?? []).length === 0}
            sx={nowrap}
          >
            {t`Clean up`}
          </Button>
        </DialogActions>
      </Dialog>
      <ImportSettingsDialog preview={importPreview} onClose={onCloseImport} />
      <MoveDialog
        move={move}
        moving={moving}
        progress={moveProgress}
        error={moveError}
        onClose={onCloseMove}
        onMove={onMove}
      />
    </>
  )
}
