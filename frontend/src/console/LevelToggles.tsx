import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Checkbox, Menu, MenuItem } from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { ChevronDown } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Level } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { formatCount } from '../mods/nexusFormat.ts'
import { countByLevel, LEVELS } from './filter.ts'
import { levelSwatch } from './levelPalette.ts'
import { useLevelNames, useShownEntries } from './logHooks.ts'
import { useConsole } from './store.ts'

// Trace and Debug are rarely wanted and very long, so they live in a menu beside the everyday levels.
const QUIET_LEVELS: Level[] = [Level.Trace, Level.Debug]

export function LevelToggles() {
  const { t, i18n } = useLingui()
  const theme = useTheme()
  const entries = useShownEntries()
  const on = useConsole((s) => s.filters.levels)
  const toggle = useConsole((s) => s.toggleLevel)
  const counts = useMemo(() => countByLevel(entries), [entries])
  const [more, setMore] = useState<HTMLElement | null>(null)
  const names = useLevelNames()
  return (
    <Box
      role="group"
      aria-label={t`Levels`}
      sx={{
        display: 'flex',
        flexWrap: 'nowrap',
        flexShrink: 0,
        p: '3px',
        gap: '2px',
        bgcolor: 'var(--mortar-overlay-30)',
        borderRadius: '8px',
      }}
    >
      {LEVELS.filter((level) => !QUIET_LEVELS.includes(level)).map((level) => {
        const pressed = on.includes(level)
        const n = counts.get(level) ?? 0
        return (
          <ButtonBase
            key={level}
            aria-pressed={pressed}
            aria-label={plural(n, {
              one: `${names[level]}: # line`,
              other: `${names[level]}: # lines`,
            })}
            onClick={() => toggle(level)}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              height: 28,
              px: 1.25,
              borderRadius: '6px',
              fontSize: 13,
              whiteSpace: 'nowrap',
              bgcolor: pressed ? 'var(--mortar-hairline-14)' : 'transparent',
              color: pressed ? 'var(--mortar-ink)' : 'var(--mortar-ink-dim)',
              '&:hover': {
                bgcolor: pressed ? 'var(--mortar-hairline-18)' : 'var(--mortar-hairline-muted)',
              },
            }}
          >
            <Box
              sx={{ width: 8, height: 8, borderRadius: '4px', bgcolor: levelSwatch(theme, level) }}
            />
            {names[level]}
            <Box component="span" sx={{ opacity: 0.75 }} aria-hidden={true}>
              {formatCount(n, i18n.locale)}
            </Box>
          </ButtonBase>
        )
      })}
      <ButtonBase
        aria-label={t`More levels`}
        aria-haspopup="menu"
        aria-expanded={more !== null}
        onClick={(e) => setMore(e.currentTarget)}
        sx={{
          height: 28,
          px: 0.75,
          borderRadius: '6px',
          color: QUIET_LEVELS.some((l) => on.includes(l))
            ? 'var(--mortar-ink)'
            : 'var(--mortar-ink-dim)',
          '&:hover': { bgcolor: 'var(--mortar-hairline-muted)' },
        }}
      >
        <ChevronDown size={14} />
      </ButtonBase>
      <Menu anchorEl={more} open={more !== null} onClose={() => setMore(null)}>
        {QUIET_LEVELS.map((level) => (
          <MenuItem
            key={level}
            role="menuitemcheckbox"
            aria-checked={on.includes(level)}
            onClick={() => toggle(level)}
          >
            <Checkbox
              size="small"
              checked={on.includes(level)}
              tabIndex={-1}
              slotProps={{ input: { 'aria-hidden': true } }}
              sx={{ p: 0, mr: 1 }}
            />
            {t`${names[level]} (${formatCount(counts.get(level) ?? 0, i18n.locale)})`}
          </MenuItem>
        ))}
      </Menu>
    </Box>
  )
}
