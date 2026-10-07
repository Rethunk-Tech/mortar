import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { MessageSquare } from 'lucide-react'
import { copyText } from './copyText.ts'
import type { ShownInfo } from './logic.ts'

const footer = {
  display: 'flex',
  alignItems: 'center',
  gap: 1.25,
  p: '14px 24px',
  borderTop: '1px solid var(--mortar-hairline-muted)',
} as const

// A message carrying the link, for pasting into a chat.
export function MessageFooter({ info, message }: { info: ShownInfo; message: string }) {
  const { t } = useLingui()
  return (
    <Box sx={footer}>
      <Box sx={{ flex: 1 }} />
      <Button
        variant="outlined"
        color="inherit"
        startIcon={<MessageSquare size={16} />}
        disabled={info.count === 0 || info.tooLarge}
        onClick={() => void copyText(message, t`Message copied`)}
        sx={{ height: 34 }}
      >
        {t`Copy as a message`}
      </Button>
    </Box>
  )
}

export function CancelFooter({ onCancel }: { onCancel: () => void }) {
  const { t } = useLingui()
  return (
    <Box sx={footer}>
      <Box sx={{ flex: 1 }} />
      <Button color="inherit" onClick={onCancel}>
        {t`Cancel`}
      </Button>
    </Box>
  )
}
