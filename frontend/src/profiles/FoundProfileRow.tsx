import { Box, Button, Typography } from '@mui/material'
import { modsLabel } from '../i18n/counts.ts'
import { space } from '../theme/density.ts'

// One profile found in another mod manager: its name, the manager, and how many mods it holds.
export function FoundProfileRow({
  name,
  source,
  mods,
  disabled,
  onClick,
}: {
  name: string
  source: string
  mods: number
  disabled?: boolean
  onClick: () => void
}) {
  return (
    <Button
      disabled={disabled ?? false}
      onClick={onClick}
      sx={{ justifyContent: 'space-between', textTransform: 'none', gap: space.pad }}
    >
      <Box sx={{ textAlign: 'left', minWidth: 0 }}>
        <Typography title={name} noWrap={true} sx={{ fontWeight: 600 }}>
          {name}
        </Typography>
        <Typography color="text.secondary" sx={{ fontSize: 12 }}>
          {source}
        </Typography>
      </Box>
      <Typography color="text.secondary" sx={{ fontSize: 13, flexShrink: 0 }}>
        {modsLabel(mods)}
      </Typography>
    </Button>
  )
}
