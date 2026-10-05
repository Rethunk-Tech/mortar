import { Box } from '@mui/material'
import type { ReactNode } from 'react'
import { DisabledReason } from './DisabledReason.tsx'

// Turns off every control inside while a source is unreachable, showing why as the tooltip; a disabled fieldset
// disables its descendants without each control needing a flag.
export function OfflineGate({ reason, children }: { reason: string; children: ReactNode }) {
  if (reason === '') {
    return children
  }
  return (
    <DisabledReason title={reason} disabled={true}>
      <Box
        component="fieldset"
        disabled={true}
        sx={{ border: 0, m: 0, p: 0, minWidth: 0, opacity: 0.5 }}
      >
        {children}
      </Box>
    </DisabledReason>
  )
}
