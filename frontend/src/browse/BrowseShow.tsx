import { useLingui } from '@lingui/react/macro'
import { Box, Button, Menu, Tooltip, Typography } from '@mui/material'
import { Filter } from 'lucide-react'
import { useState } from 'react'
import { PrefSegmented } from '../settings/PrefControls.tsx'
import { space } from '../theme/density.ts'
import { type BrowseModes, DEFAULT_MODES, type Mode, ROWS } from './browseModes.ts'

// The Mods tab's "Show" button, for what Browse does with mods in the profile, obsolete ones and broken ones.
function BrowseShow({
  modes,
  onModes,
  hasCompat,
}: {
  modes: BrowseModes
  onModes: (next: BrowseModes) => void
  hasCompat: boolean
}) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const active = ROWS.some((r) => modes[r] !== DEFAULT_MODES[r])
  const labels = {
    installed: t`Mods in this profile`,
    obsolete: t`Obsolete`,
    broken: t`Broken`,
  }
  const noCompat = t`This game has no compatibility list, so no mod is known to be broken.`
  const choices: { value: Mode; label: string }[] = [
    { value: 'off', label: t`Off` },
    { value: 'gray', label: t`Gray out` },
    { value: 'hide', label: t`Hide` },
  ]
  return (
    <>
      <Tooltip title={t`Show`}>
        <Button
          variant="outlined"
          color={active ? 'primary' : 'inherit'}
          aria-label={t`Show`}
          startIcon={<Filter size={14} />}
          aria-haspopup="menu"
          aria-expanded={anchor !== null}
          onClick={(e) => setAnchor(e.currentTarget)}
        >
          {t`Show`}
        </Button>
      </Tooltip>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={() => setAnchor(null)}>
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            gap: space.gap,
            px: space.pad,
            py: space.gap,
          }}
        >
          {ROWS.map((row) => (
            <Box key={row} sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
              <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{labels[row]}</Typography>
              <PrefSegmented
                label={labels[row]}
                value={modes[row]}
                options={choices.map((c) => ({
                  ...c,
                  ...(row === 'broken' && !hasCompat ? { unavailable: noCompat } : {}),
                }))}
                onChange={(next) => onModes({ ...modes, [row]: next as Mode })}
              />
            </Box>
          ))}
        </Box>
      </Menu>
    </>
  )
}

export { BrowseShow }
