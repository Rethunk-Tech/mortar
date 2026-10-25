import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { MessageSquare } from 'lucide-react'
import { DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { copyText } from './copyText.ts'
import type { ShownInfo } from './logic.ts'

// A message carrying the link, for pasting into a chat.
export function MessageFooter({ info, message }: { info: ShownInfo; message: string }) {
  const { t } = useLingui()
  return (
    <DialogFooter>
      <Button
        variant="outlined"
        color="inherit"
        startIcon={<MessageSquare size={16} />}
        disabled={info.count === 0 || info.tooLarge}
        onClick={() => copyText(message, t`Message copied`)}
      >
        {t`Copy as a message`}
      </Button>
    </DialogFooter>
  )
}
