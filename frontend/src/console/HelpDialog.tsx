import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { Browser, Clipboard } from '@wailsio/runtime'
import { Copy, ExternalLink, TriangleAlert } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  Log,
  Upload,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/support/service.ts'
import { paper } from '../mods/paper.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useConsole } from './store.ts'

const MONO = 'ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace'
const button = { whiteSpace: 'nowrap' } as const

function LinkRow({ link, onCopy }: { link: string; onCopy: () => void }) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
      <Box
        component="input"
        readOnly={true}
        aria-label={t`Log link`}
        value={link}
        onFocus={(e) => e.currentTarget.select()}
        sx={{
          flexGrow: 1,
          minWidth: 0,
          height: 34,
          px: 1.25,
          bgcolor: 'rgba(0,0,0,0.3)',
          border: '1px solid rgba(255,255,255,0.15)',
          borderRadius: '6px',
          color: '#ffffff',
          font: 'inherit',
          fontSize: 14,
          outline: 'none',
        }}
      />
      <Button variant="outlined" startIcon={<Copy size={16} />} onClick={onCopy} sx={button}>
        {t`Copy`}
      </Button>
      <Button
        variant="outlined"
        startIcon={<ExternalLink size={16} />}
        onClick={() => Browser.OpenURL(link).catch(reportUnexpected)}
        sx={button}
      >
        {t`Open`}
      </Button>
    </Box>
  )
}

// Shows the SMAPI log as it is on disk and uploads it to smapi.io only once the user confirms.
export function HelpDialog({ game }: { game: string }) {
  const { t } = useLingui()
  const open = useConsole((s) => s.helping)
  const setHelping = useConsole((s) => s.setHelping)
  const profile = useProfiles((s) => s.openId)
  const [log, setLog] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [link, setLink] = useState('')
  const close = () => setHelping(false)

  useEffect(() => {
    if (!open) {
      return
    }
    setLog(null)
    setLink('')
    if (!profile) {
      setLog('')
      return
    }
    Log(game, profile).then(setLog, (e: unknown) => {
      useToasts.getState().push({
        kind: 'error',
        title: t`Could not read the SMAPI log`,
        body: errorMessage(e),
      })
      setHelping(false)
    })
  }, [open, game, profile, setHelping, t])

  const copy = (text: string) =>
    Clipboard.SetText(text).then(
      () => useToasts.getState().push({ kind: 'success', title: t`Link copied` }),
      reportUnexpected,
    )

  const upload = () => {
    if (log === null || uploading) {
      return
    }
    setUploading(true)
    Upload(log)
      .then((url) => {
        setLink(url)
        return copy(url)
      })
      .catch((e: unknown) =>
        useToasts.getState().push({
          kind: 'error',
          title: t`Could not upload the log`,
          body: errorMessage(e),
        }),
      )
      .finally(() => setUploading(false))
  }

  let body: string | null = null
  if (log === '') {
    body = t`There is no SMAPI log for this profile yet. Play this profile once, then try again.`
  } else if (log === null) {
    body = t`Reading the log…`
  }

  return (
    <Dialog
      open={open}
      onClose={close}
      slotProps={{ paper: { sx: { ...paper.sx, width: 780, maxWidth: 'calc(100% - 32px)' } } }}
    >
      <DialogTitle sx={{ fontSize: 22, fontWeight: 700 }}>{t`Get help`}</DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {body ? (
          <Typography sx={{ fontSize: 14 }}>{body}</Typography>
        ) : (
          <>
            <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
              {t`Upload your SMAPI log to smapi.io, the Stardew modding community's log viewer, and share the link where you ask for help.`}
            </Typography>
            <Alert severity="warning" icon={<TriangleAlert size={16} aria-hidden={true} />}>
              {t`The log holds folder paths from this computer, which can include your user name. Anyone with the link can read it.`}
            </Alert>
            <Box
              component="textarea"
              readOnly={true}
              aria-label={t`SMAPI log`}
              value={log ?? ''}
              spellCheck={false}
              sx={{
                height: 320,
                resize: 'none',
                p: 1.5,
                bgcolor: 'rgba(0,0,0,0.5)',
                border: '1px solid rgba(255,255,255,0.15)',
                borderRadius: '6px',
                color: '#ffffff',
                fontFamily: MONO,
                fontSize: 12,
                whiteSpace: 'pre',
                outline: 'none',
              }}
            />
            {link ? <LinkRow link={link} onCopy={() => copy(link)} /> : null}
          </>
        )}
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2.5 }}>
        <Button variant="outlined" onClick={close} sx={button}>
          {link || !log ? t`Close` : t`Cancel`}
        </Button>
        {log && !link ? (
          <Button variant="contained" disabled={uploading} onClick={upload} sx={button}>
            {uploading ? t`Uploading…` : t`Upload log`}
          </Button>
        ) : null}
      </DialogActions>
    </Dialog>
  )
}
