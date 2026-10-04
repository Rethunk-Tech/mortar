import { Box, Tab, Tabs } from '@mui/material'

// A segmented switch between a few views of one page, such as Performance's Startup and In game.
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
      sx={{
        p: '4px',
        bgcolor: 'var(--mortar-overlay-30)',
        borderRadius: '8px',
        alignSelf: 'flex-start',
      }}
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
              color: 'var(--mortar-ink)',
              fontWeight: 600,
              bgcolor: 'var(--mortar-hairline-14)',
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
