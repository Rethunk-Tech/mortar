import { Box } from '@mui/material'
import { formatBytes } from '../../saves/backupFormat.ts'
import { mono } from './dataStyles.ts'

export function Row({ label, size }: { label: string; size: number }) {
  return (
    <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 2, fontSize: 13 }}>
      <Box sx={{ minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {label}
      </Box>
      <Box sx={{ ...mono, flexShrink: 0 }}>{formatBytes(size)}</Box>
    </Box>
  )
}
