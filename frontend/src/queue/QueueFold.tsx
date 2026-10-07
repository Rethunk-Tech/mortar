import { Box } from '@mui/material'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import { CoverButton } from '../shell/CoverButton.tsx'
import { space } from '../theme/density.ts'

export function Fold({
  line,
  color,
  bg,
  action,
  children,
}: {
  line: string
  color?: string
  bg: string
  action?: ReactNode
  children: ReactNode
}) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <Box
        sx={{
          position: 'relative',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: space.gap,
          minHeight: 40,
          px: '10px',
          borderRadius: '6px',
          bgcolor: bg,
          color: color ?? 'inherit',
          fontSize: 13,
        }}
      >
        <CoverButton onClick={() => setOpen(!open)} aria-expanded={open} aria-label={line} />
        <Box
          title={line}
          component="span"
          sx={{
            minWidth: 0,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            pointerEvents: 'none',
          }}
        >
          {line}
        </Box>
        {action ? <Box sx={{ position: 'relative' }}>{action}</Box> : null}
        <Box component="span" sx={{ display: 'flex', pointerEvents: 'none' }}>
          {open ? (
            <ChevronUp size={14} aria-hidden={true} />
          ) : (
            <ChevronDown size={14} aria-hidden={true} />
          )}
        </Box>
      </Box>
      {open ? children : null}
    </>
  )
}
