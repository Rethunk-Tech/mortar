import { Box, ButtonBase, Typography } from '@mui/material'
import { ChevronDown } from 'lucide-react'
import { memo } from 'react'

// A section heading in Game select; a click folds the section's games away or back.
const GroupHeading = memo(function GroupHeadingView({
  id,
  label,
  count,
  collapsed,
  onToggle,
}: {
  id: string
  label: string
  count: number
  collapsed: boolean
  onToggle: (id: string) => void
}) {
  return (
    <Typography component="h2" sx={{ m: 0 }}>
      <ButtonBase
        aria-expanded={!collapsed}
        onClick={() => onToggle(id)}
        sx={{
          width: '100%',
          justifyContent: 'flex-start',
          gap: '8px',
          px: '40px',
          pt: '14px',
          pb: '8px',
          fontSize: 12,
          fontWeight: 700,
          letterSpacing: '0.08em',
          textTransform: 'uppercase',
          color: 'var(--mortar-ink-sec)',
          '&:hover': { color: 'text.primary' },
        }}
      >
        <ChevronDown
          size={14}
          aria-hidden={true}
          style={{ transform: collapsed ? 'rotate(-90deg)' : 'none' }}
        />
        {label}
        <Box component="span" sx={{ fontWeight: 400, letterSpacing: 0 }}>
          {count}
        </Box>
      </ButtonBase>
    </Typography>
  )
})

export { GroupHeading }
