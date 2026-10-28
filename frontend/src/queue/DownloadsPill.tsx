import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, LinearProgress } from '@mui/material'
import { Download } from 'lucide-react'
import { memo } from 'react'
import { compact } from '../game/compact.ts'
import { closeTitleMenu } from '../shell/titleMenus.ts'
import { space } from '../theme/density.ts'
import { useQueue } from './store.ts'
import { pillProgress } from './totals.ts'

// The title bar's downloads button: there only while the queue has items under way, with how many and how far along.
// Memoised: the title bar re-renders on every navigation, and this one does not need to.
export const DownloadsPill = memo(function DownloadsPillView() {
  const { t } = useLingui()
  const items = useQueue((s) => s.state.items)
  const setOpen = useQueue((s) => s.setOpen)
  const progress = pillProgress(items)
  if (!progress) {
    return null
  }
  const { active, percent } = progress
  const downloading = plural({ count: active }, { one: '# downloading', other: '# downloading' })
  return (
    <ButtonBase
      onClick={() => {
        closeTitleMenu()
        setOpen(true)
      }}
      aria-label={t`Downloads: ${downloading} ${percent}%`}
      sx={{
        '--wails-draggable': 'no-drag',
        gap: space.gap,
        height: space.control,
        px: '12px',
        borderRadius: '8px',
        fontFamily: 'inherit',
        fontSize: 'medium.fontSize',
        whiteSpace: 'nowrap',
        color: 'inherit',
        border: '1px solid var(--mortar-hairline-12)',
        '&:hover': { bgcolor: 'var(--mortar-hairline-faint)' },
        [compact]: { px: '8px' },
      }}
    >
      <Box component="span" sx={{ display: 'flex', color: 'var(--mortar-accent-ink)' }}>
        <Download size={16} aria-hidden={true} />
      </Box>
      <Box component="span" sx={{ [compact]: { display: 'none' } }}>
        {downloading}
      </Box>
      <LinearProgress
        variant="determinate"
        value={percent}
        aria-hidden={true}
        sx={{
          width: 72,
          height: 4,
          borderRadius: '2px',
          bgcolor: 'var(--mortar-hairline-12)',
          [compact]: { display: 'none' },
        }}
      />
      <Box component="span" sx={{ fontSize: 'small.fontSize', color: 'var(--mortar-ink-90)' }}>
        {`${percent}%`}
      </Box>
    </ButtonBase>
  )
})
