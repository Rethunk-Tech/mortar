import { Box, Button, Checkbox, Typography } from '@mui/material'
import { modsLabel } from '../i18n/counts.ts'
import { space } from '../theme/density.ts'

// One profile found in another mod manager: its name, the manager, and how many mods it holds.
export function FoundProfileRow({
  name,
  source,
  mods,
  disabled,
  check,
  onClick,
}: {
  name: string
  source: string
  mods: number
  disabled?: boolean
  // Shows a checkbox that picks the profile for a batch import without opening it.
  check?: { checked: boolean; onChange: (checked: boolean) => void }
  onClick: () => void
}) {
  const row = (
    <Button
      disabled={disabled ?? false}
      onClick={onClick}
      sx={{ justifyContent: 'space-between', textTransform: 'none', gap: space.pad }}
    >
      <Box sx={{ textAlign: 'left', minWidth: 0 }}>
        <Typography title={name} noWrap={true} sx={{ fontWeight: 600 }}>
          {name}
        </Typography>
        <Typography color="text.secondary" sx={{ fontSize: 'small.fontSize' }}>
          {source}
        </Typography>
      </Box>
      <Typography color="text.secondary" sx={{ fontSize: 'base.fontSize', flexShrink: 0 }}>
        {modsLabel(mods)}
      </Typography>
    </Button>
  )
  if (!check) {
    return row
  }
  return (
    <Box sx={{ display: 'flex', alignItems: 'center' }}>
      <Checkbox
        checked={check.checked}
        onChange={(event) => check.onChange(event.target.checked)}
        slotProps={{ input: { 'aria-label': name } }}
      />
      <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>{row}</Box>
    </Box>
  )
}
