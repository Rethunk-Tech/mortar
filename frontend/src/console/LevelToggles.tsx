import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Checkbox, Menu, MenuItem } from '@mui/material'
import { ChevronDown } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Level } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import { countByLevel, LEVELS } from './filter.ts'
import { useShownEntries } from './logHooks.ts'
import { useConsole } from './store.ts'

const dots: Record<Level, string> = {
  [Level.$zero]: 'transparent',
  [Level.Trace]: '#9a9aa6',
  [Level.Debug]: '#b4b4c0',
  [Level.Info]: '#ececf0',
  [Level.Warn]: '#F3B416',
  [Level.Error]: '#ff6b5f',
  [Level.Alert]: '#c792ea',
}

// Trace and Debug are rarely wanted and very long, so they live in a menu beside the everyday levels.
const QUIET_LEVELS: Level[] = [Level.Trace, Level.Debug]
const compactCount = new Intl.NumberFormat(undefined, { notation: 'compact' })

export function LevelToggles() {
  const { t } = useLingui()
  const entries = useShownEntries()
  const on = useConsole((s) => s.filters.levels)
  const toggle = useConsole((s) => s.toggleLevel)
  const counts = useMemo(() => countByLevel(entries), [entries])
  const [more, setMore] = useState<HTMLElement | null>(null)
  const names: Record<Level, string> = {
    [Level.$zero]: '',
    [Level.Trace]: t`Trace`,
    [Level.Debug]: t`Debug`,
    [Level.Info]: t`Info`,
    [Level.Warn]: t`Warn`,
    [Level.Error]: t`Error`,
    [Level.Alert]: t`Alert`,
  }
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
            aria-label={t`${names[level]}: ${n} lines`}
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
            <Box sx={{ width: 8, height: 8, borderRadius: '4px', bgcolor: dots[level] }} />
            {names[level]}
            <Box component="span" sx={{ opacity: 0.75 }} aria-hidden={true}>
              {compactCount.format(n)}
            </Box>
          </ButtonBase>
        )
      })}
      <ButtonBase
        aria-label={t`More levels`}
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
          <MenuItem key={level} onClick={() => toggle(level)}>
            <Checkbox size="small" checked={on.includes(level)} sx={{ p: 0, mr: 1 }} />
            {t`${names[level]} (${compactCount.format(counts.get(level) ?? 0)})`}
          </MenuItem>
        ))}
      </Menu>
    </Box>
  )
}
