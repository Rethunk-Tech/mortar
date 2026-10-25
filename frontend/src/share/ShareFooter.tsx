import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { Copy, MessageSquare, Save } from 'lucide-react'
import { useState } from 'react'
import { DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { copyText } from './copyText.ts'
import { PreviewDialog } from './PreviewDialog.tsx'
import type { useShareBuild } from './useShareBuild.ts'

// Share's Mortar footer: what they will see on the left, the link as a chat message, and the main action, Copy link
// or Save… for a file.
export function MortarFooter({
  built,
  message,
}: {
  built: ReturnType<typeof useShareBuild>
  message: string
}) {
  const { t } = useLingui()
  const [previewing, setPreviewing] = useState(false)
  const { info, format, saving, saveFile } = built
  if (!info) {
    return null
  }
  const empty = info.count === 0
  return (
    <DialogFooter
      aside={
        <>
          <Button
            variant="text"
            disabled={info.tooLarge}
            onClick={() => setPreviewing(true)}
            sx={{ ml: -1 }}
          >
            {t`Preview what they see`}
          </Button>
          <PreviewDialog info={info} open={previewing} onClose={() => setPreviewing(false)} />
        </>
      }
    >
      <Button
        startIcon={<MessageSquare size={16} />}
        disabled={empty || info.tooLarge}
        onClick={() => copyText(message, t`Message copied`)}
      >
        {t`Copy as a message`}
      </Button>
      {format === 'link' ? (
        <Button
          variant="contained"
          startIcon={<Copy size={16} />}
          disabled={empty || info.tooLarge}
          onClick={() => copyText(info.web, t`Link copied`)}
        >
          {t`Copy link`}
        </Button>
      ) : (
        <Button
          variant="contained"
          startIcon={<Save size={16} />}
          disabled={saving || empty}
          onClick={saveFile}
        >
          {t`Save…`}
        </Button>
      )}
    </DialogFooter>
  )
}
