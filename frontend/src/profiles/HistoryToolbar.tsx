import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { GitCompare, RotateCcw, ShieldCheck } from 'lucide-react'

export function HistoryToolbar({
  busy,
  canRestore,
  comparing,
  picked,
  onMark,
  onRestore,
  onToggleCompare,
  onCompare,
}: {
  busy: boolean
  // False until a point in the history is marked known good.
  canRestore: boolean
  comparing: boolean
  picked: number
  onMark: () => void
  onRestore: () => void
  onToggleCompare: () => void
  onCompare: () => void
}) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', gap: 1, mb: 1.5, flexWrap: 'wrap', alignItems: 'center' }}>
      <Button
        size="small"
        variant="outlined"
        startIcon={<ShieldCheck size={16} />}
        disabled={busy}
        onClick={onMark}
      >
        {t`Mark current as known good`}
      </Button>
      <Button
        size="small"
        variant="outlined"
        startIcon={<RotateCcw size={16} />}
        disabled={busy || !canRestore}
        onClick={onRestore}
      >
        {t`Restore known good`}
      </Button>
      {canRestore ? null : (
        <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
          {t`Mark a known good point first.`}
        </Typography>
      )}
      <Box sx={{ flex: 1 }} />
      {comparing ? (
        <>
          <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
            {t`Pick two entries (${picked} of 2)`}
          </Typography>
          <Button size="small" variant="contained" disabled={picked !== 2} onClick={onCompare}>
            {t`Compare`}
          </Button>
        </>
      ) : null}
      <Button
        size="small"
        variant={comparing ? 'text' : 'outlined'}
        startIcon={<GitCompare size={16} />}
        aria-pressed={comparing}
        disabled={busy}
        onClick={onToggleCompare}
      >
        {comparing ? t`Stop comparing` : t`Compare two…`}
      </Button>
    </Box>
  )
}
