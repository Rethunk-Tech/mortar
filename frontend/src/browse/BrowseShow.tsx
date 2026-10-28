import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, FormControlLabel, Popover, Typography } from '@mui/material'
import { Filter } from 'lucide-react'
import { useState } from 'react'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { PrefSegmented } from '../settings/PrefControls.tsx'
import { saveSetting } from '../settings/persist.ts'
import { useSettings } from '../settings/store.ts'
import { Tooltip } from '../shell/Tooltip.tsx'
import { space } from '../theme/density.ts'
import { type BrowseModes, DEFAULT_MODES, type Mode, ROWS } from './browseModes.ts'
import { CARD_STATS, type CardStat, formatCardStats, parseCardStats } from './stats.ts'

// The Mods tab's "Show" button, for what Browse does with mods in the profile, obsolete ones and broken ones, and
// which stats its cards show.
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
  const cardStats = parseCardStats(useSettings((s) => s.browseCardStats))
  const statLabels: Record<CardStat, string> = {
    popularity: t`Endorsements, likes or stars`,
    downloads: t`Downloads`,
    size: t`Size`,
    version: t`Version`,
    updated: t`Last updated`,
  }
  const toggleStat = (stat: CardStat, on: boolean) => {
    const next = on ? [...cardStats, stat] : cardStats.filter((s) => s !== stat)
    saveSetting(() => SetByKey('browseCardStats', formatCardStats(next), ''))
  }
  const choices: { value: Mode; label: string }[] = [
    { value: 'off', label: t`Off` },
    { value: 'flag', label: t`Flag` },
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
      {/* A panel of settings, not a list of actions, so a popover rather than a menu. */}
      <Popover open={anchor !== null} anchorEl={anchor} onClose={() => setAnchor(null)}>
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
              <Typography sx={{ fontSize: 'base.fontSize', fontWeight: 600 }}>
                {labels[row]}
              </Typography>
              <PrefSegmented
                label={labels[row]}
                value={modes[row]}
                options={choices
                  // An installed mod already wears its Added chip; Flag is for warnings.
                  .filter((c) => row !== 'installed' || c.value !== 'flag')
                  .map((c) => ({
                    ...c,
                    ...(row === 'broken' && !hasCompat ? { unavailable: noCompat } : {}),
                  }))}
                onChange={(next) => onModes({ ...modes, [row]: next as Mode })}
              />
            </Box>
          ))}
          <Box sx={{ display: 'flex', flexDirection: 'column' }}>
            <Typography
              sx={{ fontSize: 'base.fontSize', fontWeight: 600 }}
            >{t`Card details`}</Typography>
            {CARD_STATS.map((stat) => (
              <FormControlLabel
                key={stat}
                label={statLabels[stat]}
                control={
                  <Checkbox
                    size="small"
                    checked={cardStats.includes(stat)}
                    onChange={(e) => toggleStat(stat, e.target.checked)}
                  />
                }
              />
            ))}
          </Box>
        </Box>
      </Popover>
    </>
  )
}

export { BrowseShow }
