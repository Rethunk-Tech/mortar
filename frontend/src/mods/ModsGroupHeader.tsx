import { Box, ButtonBase, Typography } from '@mui/material'
import { ChevronDown, ChevronRight } from 'lucide-react'

export function ModsGroupHeader({
  label,
  count,
  open,
  onToggle,
  hint,
}: {
  label: string
  count: number
  open: boolean
  onToggle: () => void
  hint?: string
}) {
  return (
    <ButtonBase
      onClick={onToggle}
      aria-expanded={open}
      title={hint}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 0.75,
        width: '100%',
        px: 2,
        py: 0.75,
        justifyContent: 'flex-start',
        textAlign: 'left',
        fontFamily: 'inherit',
        color: 'inherit',
        bgcolor: 'rgba(255,255,255,0.04)',
        borderBottom: '1px solid rgba(255,255,255,0.08)',
      }}
    >
      {open ? (
        <ChevronDown size={14} aria-hidden={true} />
      ) : (
        <ChevronRight size={14} aria-hidden={true} />
      )}
      <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{label}</Typography>
      <Box component="span" sx={{ fontSize: 12, color: 'text.secondary' }}>
        {count}
      </Box>
    </ButtonBase>
  )
}
