import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Tooltip } from '@mui/material'
import { compact } from '../game/compact.ts'
import { useSettings } from '../settings/store.ts'
import { healthView } from './badgeDisplay.ts'
import type { Counts } from './badges.ts'

const pill = {
  flexShrink: 0,
  ml: 0.5,
  px: '7px',
  py: '1px',
  borderRadius: '10px',
  color: '#1b1a17',
  fontSize: 12,
  fontWeight: 700,
}

const sidebarPill = {
  ...pill,
  [compact]: {
    position: 'absolute' as const,
    top: 1,
    right: 1,
    ml: 0,
    px: '4px',
    fontSize: 10,
  },
  '[data-collapsed="true"] &': {
    position: 'absolute' as const,
    top: 1,
    right: 1,
    ml: 0,
    px: '4px',
    fontSize: 10,
  },
}

export function ProfileHealth({
  counts,
  sidebar = false,
}: {
  counts?: Counts | undefined
  sidebar?: boolean
}) {
  const { t } = useLingui()
  const mode = useSettings((s) => s.sidebarBadges)
  const view = healthView(counts, mode, {
    missing: (n) =>
      t`${plural(n, { one: '# missing requirement', other: '# missing requirements' })}`,
    problems: (n) => t`${plural(n, { one: '# problem', other: '# problems' })}`,
    updates: (n) => t`${plural(n, { one: '# update', other: '# updates' })}`,
  })
  if (view.tone === 'none') {
    return null
  }
  return (
    <Tooltip title={view.tooltip} disableInteractive={true}>
      <Box
        component="span"
        role="img"
        aria-label={view.tooltip}
        sx={{
          ...(sidebar ? sidebarPill : pill),
          bgcolor: view.tone === 'red' ? 'error.main' : 'warning.main',
        }}
      >
        {view.value}
      </Box>
    </Tooltip>
  )
}
