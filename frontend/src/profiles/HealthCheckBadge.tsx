import { plural } from '@lingui/core/macro'
import { Box, ButtonBase, Tooltip } from '@mui/material'
import { ShieldAlert } from 'lucide-react'
import { useEffect, useState } from 'react'
import { pill } from '../mods/pills.ts'
import { HealthDialog } from './HealthDialog.tsx'
import { useHealthBadges } from './healthBadges.ts'

// HealthCheckBadge shows that the profile's latest health check found something, and opens the check on click.
export function HealthCheckBadge({ game, profileId }: { game: string; profileId: string }) {
  const count = useHealthBadges((s) => s.byProfile[profileId] ?? 0)
  const load = useHealthBadges((s) => s.load)
  const [open, setOpen] = useState(false)
  useEffect(() => load(game), [game, load])
  const label = plural(count, { one: '# thing to repair', other: '# things to repair' })
  return (
    <>
      {count > 0 ? (
        <Tooltip title={label} disableInteractive={true}>
          <ButtonBase
            aria-label={label}
            onClick={() => setOpen(true)}
            sx={{
              ...pill,
              display: 'inline-flex',
              alignItems: 'center',
              gap: '3px',
              bgcolor: 'warning.main',
            }}
          >
            <ShieldAlert size={12} />
            <Box component="span">{count}</Box>
          </ButtonBase>
        </Tooltip>
      ) : null}
      <HealthDialog profileId={profileId} open={open} onClose={() => setOpen(false)} />
    </>
  )
}
