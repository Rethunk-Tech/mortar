import { Box, Button, Collapse } from '@mui/material'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { type ReactNode, useState } from 'react'

export function Fold({ title, children }: { title: string; children: ReactNode }) {
  const [shown, setShown] = useState(false)
  const Icon = shown ? ChevronDown : ChevronRight
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Button
        size="small"
        onClick={() => setShown(!shown)}
        startIcon={<Icon size={14} aria-hidden={true} />}
        aria-expanded={shown}
        sx={{
          whiteSpace: 'nowrap',
          fontSize: 12,
          color: 'text.secondary',
          fontWeight: 700,
          textTransform: 'none',
          alignSelf: 'flex-start',
          px: 0.5,
        }}
      >
        {title}
      </Button>
      <Collapse in={shown} unmountOnExit={true}>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, pl: 1 }}>{children}</Box>
      </Collapse>
    </Box>
  )
}
