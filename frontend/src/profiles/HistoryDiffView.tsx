import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { HistoryDiff } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { space } from '../theme/density.ts'
import { diffLines } from './historyDiff.ts'

export function HistoryDiffView({
  diff,
  aLabel,
  bLabel,
  onRestoreA,
  onRestoreB,
  busy,
}: {
  diff: HistoryDiff
  aLabel: string
  bLabel: string
  onRestoreA: () => void
  onRestoreB: () => void
  busy: boolean
}) {
  const { t } = useLingui()
  const lines = diffLines(diff)
  return (
    <Box sx={{ mb: 2, p: space.pad, bgcolor: 'var(--mortar-paper-78)', borderRadius: '6px' }}>
      <Typography sx={{ fontWeight: 600, fontSize: 13 }}>{t`Snapshot diff`}</Typography>
      {lines.length === 0 ? (
        <Typography
          sx={{ mt: 1, fontSize: 13, color: 'text.secondary' }}
        >{t`No differences.`}</Typography>
      ) : (
        <Box component="ul" sx={{ m: 0, mt: 1, pl: space.pad, fontSize: 13 }}>
          {lines.map((line) => (
            <li key={line}>{line}</li>
          ))}
        </Box>
      )}
      <Box sx={{ display: 'flex', gap: space.gap, mt: 1.5, flexWrap: 'wrap' }}>
        <Button size="small" disabled={busy} onClick={onRestoreA}>
          {t`Restore ${{ label: aLabel }}`}
        </Button>
        <Button size="small" disabled={busy} onClick={onRestoreB}>
          {t`Restore ${{ label: bLabel }}`}
        </Button>
      </Box>
    </Box>
  )
}
