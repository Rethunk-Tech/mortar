import { Box } from '@mui/material'
import { accent } from '../paper.ts'
import { MEDIUM, MONO } from './constants.ts'

export function Version({ children, isNew }: { children: string; isNew?: boolean }) {
  return (
    <Box
      component="span"
      title={children}
      sx={{
        px: 1,
        py: '3px',
        borderRadius: '4px',
        display: 'inline-block',
        maxWidth: 150,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        verticalAlign: 'middle',
        fontFamily: MONO,
        fontSize: 13,
        fontWeight: isNew ? MEDIUM : 'normal',
        color: isNew ? 'primary.main' : 'text.primary',
        bgcolor: isNew ? accent.chip : 'rgba(255,255,255,0.08)',
      }}
    >
      {children}
    </Box>
  )
}
