import { Box, ButtonBase, Tooltip } from '@mui/material'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { hasSourceLogo } from '../brand/sources/sourceIcons.ts'

// One badge per source the mod is on; the filled one is where Add installs from.
export function SourceBadges({
  sources,
  picked,
  names,
  onPick,
}: {
  sources: string[]
  picked: string
  names: Map<string, string>
  onPick: (source: string) => void
}) {
  if (sources.length < 2) {
    return null
  }
  return (
    <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', pt: 0.25 }}>
      {sources.map((id) => {
        const name = names.get(id) ?? id
        const mark = hasSourceLogo(id) ? <SourceLogo id={id} size={14} /> : <span>{name}</span>
        return (
          <Tooltip key={id} title={name}>
            <ButtonBase
              aria-label={name}
              aria-pressed={id === picked}
              onClick={() => onPick(id)}
              sx={{
                minWidth: 28,
                height: 24,
                px: 0.5,
                borderRadius: '6px',
                fontSize: 12,
                border: '1px solid',
                borderColor: id === picked ? 'var(--mortar-ink-dim-60)' : 'var(--mortar-hairline)',
                bgcolor: id === picked ? 'var(--mortar-hairline-16)' : 'transparent',
                color: 'text.secondary',
                '&:hover': { bgcolor: 'var(--mortar-hairline-muted)' },
              }}
            >
              {mark}
            </ButtonBase>
          </Tooltip>
        )
      })}
    </Box>
  )
}
