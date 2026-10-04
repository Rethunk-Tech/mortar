import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { Copy, ExternalLink, TriangleAlert } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { RunLog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import {
  Log,
  Upload,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'
import { openPage } from '../mods/menu.ts'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { MONO } from '../theme/theme.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { anonymize } from './anonymize.ts'
import { shareLogConfirm } from './shareLog.ts'
import { useConsole } from './store.ts'

const button = { whiteSpace: 'nowrap' } as const

function LinkRow({ link, onCopy }: { link: string; onCopy: () => void }) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
      <TextField
        size="small"
        value={link}
        sx={{ flexGrow: 1, minWidth: 0 }}
        slotProps={{
          htmlInput: {
            readOnly: true,
            'aria-label': t`Log link`,
            onFocus: (e: React.FocusEvent<HTMLInputElement>) => e.currentTarget.select(),
          },
        }}
      />
      <Button variant="outlined" startIcon={<Copy size={16} />} onClick={onCopy} sx={button}>
        {t`Copy`}
      </Button>
      <Button
        variant="outlined"
        startIcon={<ExternalLink size={16} />}
        onClick={() => openPage(link)}
        sx={button}
      >
        {t`Open`}
      </Button>
    </Box>
  )
}

function HideUserName({
  checked,
  onChange,
}: {
  checked: boolean
  onChange: (value: boolean) => void
}) {
  const { t } = useLingui()
  return (
    <FormControlLabel
      control={<Switch checked={checked} onChange={(_, value) => onChange(value)} />}
      label={t`Hide my user name`}
    />
  )
}

function HelpLog({
  log,
  link,
  hideUserName,
  setHideUserName,
  copy,
}: {
  log: string
  link: string
  hideUserName: boolean
  setHideUserName: (value: boolean) => void
  copy: (text: string) => void
}) {
  const { t, i18n } = useLingui()
  return (
    <>
      <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
        {shareLogConfirm(i18n, new TextEncoder().encode(log).length)}
      </Typography>
      <Alert severity="warning" icon={<TriangleAlert size={16} aria-hidden={true} />}>
        {t`The log holds folder paths from this computer, which can include your user name. Anyone with the link can read it.`}
      </Alert>
      <TextField
        multiline={true}
        rows={15}
        value={log}
        slotProps={{
          htmlInput: {
            readOnly: true,
            spellCheck: false,
            'aria-label': t`SMAPI log`,
            sx: { fontFamily: MONO, fontSize: 12, whiteSpace: 'pre', overflowX: 'auto' },
          },
        }}
      />
      <HideUserName checked={hideUserName} onChange={setHideUserName} />
      {link ? <LinkRow link={link} onCopy={() => copy(link)} /> : null}
    </>
  )
}

// Shows the SMAPI log as it is on disk and uploads it to smapi.io only once the user confirms.
export function HelpDialog({ game }: { game: string }) {
  const { t } = useLingui()
  const open = useConsole((s) => s.helping)
  const setHelping = useConsole((s) => s.setHelping)
  const profile = useConsole((s) => s.shown.profile)
  const viewingRun = useConsole((s) => s.viewingRun)
  const openFor = useRef({ game, profile })
  openFor.current = { game, profile }
  const uploadGen = useRef(0)
  const [log, setLog] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [link, setLink] = useState('')
  const [hideUserName, setHideUserName] = useState(true)
  const close = () => setHelping(false)
  useEffect(() => {
    uploadGen.current += 1
    if (!open) {
      return
    }
    setLog(null)
    setLink('')
    if (!profile) {
      setLog('')
      return
    }
    let live = true
    const read = viewingRun ? RunLog(game, profile, viewingRun) : Log(game, profile)
    read.then(
      (text) => {
        if (live) {
          setLog(text)
        }
      },
      (e: unknown) => {
        if (!live) {
          return
        }
        reportError(t`Could not read the SMAPI log`)(e)
        setHelping(false)
      },
    )
    return () => {
      live = false
    }
  }, [open, game, profile, viewingRun, setHelping, t])

  const copy = (text: string) =>
    Clipboard.SetText(text).then(
      () => useToasts.getState().push({ kind: 'success', title: t`Link copied` }),
      reportUnexpected,
    )

  const upload = () => {
    const token = uploadGen.current
    if (log === null || uploading) {
      return
    }
    setUploading(true)
    Upload(hideUserName ? anonymize(log) : log)
      .then((url) => {
        if (uploadGen.current !== token) {
          return
        }
        setLink(url)
        return copy(url)
      })
      .catch((e: unknown) => {
        if (uploadGen.current !== token) {
          return
        }
        reportError(t`Could not upload the log`)(e)
      })
      .finally(() => {
        if (uploadGen.current === token) {
          setUploading(false)
        }
      })
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
      onClose={uploading ? undefined : close}
      slotProps={{ paper: { sx: { width: 780, maxWidth: 'calc(100% - 32px)' } } }}
    >
      <DialogTitle sx={{ fontSize: 22, fontWeight: 700 }}>{t`Share log…`}</DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {body ? (
          <>
            {log === null ? <LoadingRow>{body}</LoadingRow> : null}
            {log === null ? null : <Typography sx={{ fontSize: 14 }}>{body}</Typography>}
          </>
        ) : (
          <HelpLog
            log={log ?? ''}
            link={link}
            hideUserName={hideUserName}
            setHideUserName={setHideUserName}
            copy={copy}
          />
        )}
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2.5 }}>
        <Button disabled={uploading} onClick={close} sx={button}>
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
