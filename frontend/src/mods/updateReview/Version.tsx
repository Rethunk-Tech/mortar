import { Box } from '@mui/material'
import { space } from '../../theme/density.ts'
import { MONO } from '../../theme/theme.ts'
import { accent } from '../paper.ts'
import { MEDIUM } from './constants.ts'

export function Version({ children, isNew }: { children: string; isNew?: boolean }) {
  return (
    <Box
      component="span"
      title={children}
      sx={{
        px: space.gap,
        py: '3px',
        borderRadius: '4px',
        display: 'inline-block',
        maxWidth: 260,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        verticalAlign: 'middle',
        fontFamily: MONO,
        fontSize: 13,
        fontWeight: isNew ? MEDIUM : 'normal',
        color: isNew ? 'primary.main' : 'text.primary',
        bgcolor: isNew ? accent.chip : 'var(--mortar-hairline-muted)',
      }}
    >
      {children}
    </Box>
  )
}
