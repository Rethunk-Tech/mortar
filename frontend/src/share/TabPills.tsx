import { Box, Tab, Tabs } from '@mui/material'

// The segmented tab switch of the Share and Import dialogs.
export function TabPills<T extends string>({
  value,
  onChange,
  options,
  label,
}: {
  value: T
  onChange: (value: T) => void
  options: { value: T; label: string }[]
  label: string
}) {
  return (
    <Box
      sx={{ p: '4px', bgcolor: 'rgba(0,0,0,0.3)', borderRadius: '8px', alignSelf: 'flex-start' }}
    >
      <Tabs
        value={value}
        onChange={(_, next: T) => onChange(next)}
        aria-label={label}
        slotProps={{ indicator: { sx: { display: 'none' } } }}
        sx={{
          minHeight: 36,
          '& .MuiTabs-flexContainer': { gap: '4px' },
          '& .MuiTab-root': {
            minHeight: 36,
            height: 36,
            minWidth: 0,
            px: 2,
            borderRadius: '6px',
            fontSize: 14,
            fontWeight: 400,
            color: 'text.secondary',
            '&.Mui-selected': {
              color: '#ffffff',
              fontWeight: 600,
              bgcolor: 'rgba(255,255,255,0.14)',
            },
          },
        }}
      >
        {options.map((o) => (
          <Tab key={o.value} value={o.value} label={o.label} disableRipple={true} />
        ))}
      </Tabs>
    </Box>
  )
}
