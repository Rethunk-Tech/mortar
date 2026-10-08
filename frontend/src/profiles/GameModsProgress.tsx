import { useLingui } from '@lingui/react/macro'
import { Box, LinearProgress, Typography } from '@mui/material'
import type { GameModsProgress } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

const PERCENT = 100

/** A determinate bar and "Importing N of M · name" (or "Moving…") while folders are copied in. */
export function GameModsProgressLine({
  progress,
  moving,
}: {
  progress: GameModsProgress | null
  moving: boolean
}) {
  const { t } = useLingui()
  const n = progress === null ? 0 : progress.done + 1
  const total = progress?.total ?? 0
  const verb = moving ? t`Moving` : t`Importing`
  return (
    <Box sx={{ minWidth: 320 }} role="status">
      <LinearProgress
        variant="determinate"
        value={progress === null || total === 0 ? 0 : (progress.done / total) * PERCENT}
        sx={{ my: 1.5 }}
      />
      <Typography sx={{ fontSize: 14, color: 'text.secondary', overflowWrap: 'anywhere' }}>
        {progress === null ? `${verb}…` : t`${verb} ${n} of ${total} · ${progress.name}`}
      </Typography>
    </Box>
  )
}
