import { useLingui } from '@lingui/react/macro'
import { Box, Dialog, IconButton, Typography } from '@mui/material'
import { X } from 'lucide-react'
import { useEffect } from 'react'
import { paper } from '../mods/paper.ts'
import { useProfiles } from '../profiles/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { ImportFooter } from './ImportFooter.tsx'
import { ImportInput } from './ImportInput.tsx'
import { Tiles } from './ImportPreview.tsx'
import { summarize } from './logic.ts'
import { type ImportRequest, useImportDialog } from './store.ts'
import { TabPills } from './TabPills.tsx'
import { useImportFlow } from './useImportFlow.ts'

const dialogSx = {
  ...paper.sx,
  width: 'min(1180px, calc(100% - 48px))',
  height: 'min(680px, calc(100% - 48px))',
  overflow: 'hidden',
  borderRadius: '10px',
}

function Body({ request }: { request: ImportRequest }) {
  const { t } = useLingui()
  const close = useImportDialog((s) => s.close)
  const game = useProfiles((s) => s.game?.id ?? 'stardew')
  const targetName = useProfiles((s) => s.profiles.find((p) => p.id === request.profileId)?.name)
  const signedIn = useNexus((s) => s.signedIn)
  const flow = useImportFlow(game, request.profileId, close)
  const { setTab, setText, previewLink, previewFile } = flow

  useEffect(() => {
    setTab(request.tab)
    if (request.seed && request.tab === 'link') {
      setText(request.seed)
      previewLink(request.seed).catch(reportUnexpected)
    } else if (request.seed) {
      previewFile(request.seed).catch(reportUnexpected)
    }
  }, [request, setTab, setText, previewLink, previewFile])

  const { preview } = flow
  return (
    <Box
      role="dialog"
      aria-label={t`Import profile`}
      sx={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 2,
          p: '16px 20px',
          borderBottom: '1px solid rgba(255,255,255,0.1)',
        }}
      >
        <Typography sx={{ fontSize: 20, fontWeight: 700, whiteSpace: 'nowrap' }}>
          {t`Import profile from…`}
        </Typography>
        <TabPills
          upper={true}
          value={flow.tab}
          onChange={(next) => {
            flow.reset()
            setTab(next)
          }}
          label={t`Where the profile comes from`}
          options={[
            { value: 'link', label: t`Link` },
            { value: 'file', label: t`.mortar file` },
          ]}
        />
        <Box sx={{ flex: 1 }} />
        <IconButton aria-label={t`Close`} onClick={flow.dismiss}>
          <X size={18} />
        </IconButton>
      </Box>
      <Box sx={{ flex: 1, minHeight: 0, overflowY: 'auto', p: 2 }}>
        {preview ? (
          <Tiles mods={preview.mods} excluded={flow.excluded} onToggle={flow.toggle} />
        ) : (
          <ImportInput flow={flow} />
        )}
      </Box>
      {preview ? (
        <ImportFooter
          flow={flow}
          preview={preview}
          summary={summarize(preview.mods, flow.excluded)}
          targetName={targetName ?? ''}
          signedIn={signedIn}
        />
      ) : null}
    </Box>
  )
}

export function ImportDialog() {
  const request = useImportDialog((s) => s.request)
  const dismiss = useImportDialog((s) => s.close)
  return (
    <Dialog
      open={request !== null}
      onClose={dismiss}
      maxWidth={false}
      slotProps={{ paper: { ...paper, sx: dialogSx } }}
    >
      {request ? <Body key={request.run} request={request} /> : null}
    </Dialog>
  )
}
