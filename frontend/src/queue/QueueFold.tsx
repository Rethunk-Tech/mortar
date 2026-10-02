import { Box } from '@mui/material'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { type ReactNode, useState } from 'react'

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
        component="button"
        onClick={() => setOpen(!open)}
        aria-expanded={open}
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 1,
          minHeight: 40,
          px: '10px',
          border: 0,
          borderRadius: '6px',
          bgcolor: bg,
          color: color ?? 'inherit',
          fontFamily: 'inherit',
          fontSize: 13,
          textAlign: 'left',
          cursor: 'pointer',
        }}
      >
        <Box
          component="span"
          sx={{ minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
        >
          {line}
        </Box>
        {action ? <Box onClick={(e) => e.stopPropagation()}>{action}</Box> : null}
        {open ? (
          <ChevronUp size={14} aria-hidden={true} />
        ) : (
          <ChevronDown size={14} aria-hidden={true} />
        )}
      </Box>
      {open ? children : null}
    </>
  )
}
