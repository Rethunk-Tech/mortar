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
import { Copy, ExternalLink, TriangleAlert } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import {
  RunLog,
  Runs,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import {
  Log,
  Upload,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'
import { openPage } from '../mods/menu.ts'
import { useProfileLoader } from '../profiles/store.ts'
import { copyText } from '../share/copyText.ts'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { reportError } from '../toasts/report.ts'
import { anonymize } from './anonymize.ts'
import { pasteLogConfirm, shareLogConfirm, shareLogText } from './shareLog.ts'
import { useConsole } from './store.ts'

const button = { whiteSpace: 'nowrap' } as const

function LinkRow({ link, onCopy }: { link: string; onCopy: () => void }) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
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
  paste,
  hideUserName,
  setHideUserName,
  copy,
}: {
  log: string
  link: string
  paste: string
  hideUserName: boolean
  setHideUserName: (value: boolean) => void
  copy: (text: string) => void
}) {
  const { t, i18n } = useLingui()
  return (
    <>
      <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
        {paste
          ? pasteLogConfirm(i18n, new TextEncoder().encode(log).length, paste)
          : shareLogConfirm(i18n, new TextEncoder().encode(log).length)}
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
            'aria-label': t`Game log`,
            sx: { fontFamily: MONO, fontSize: 12, whiteSpace: 'pre', overflowX: 'auto' },
          },
        }}
      />
      <HideUserName checked={hideUserName} onChange={setHideUserName} />
      {link ? <LinkRow link={link} onCopy={() => copy(link)} /> : null}
    </>
  )
}

// Shows the run's log as it is on disk and, only once the user confirms, uploads it to smapi.io or, for a loader with
// a paste site, copies it and opens that site.
export function HelpDialog({ game }: { game: string }) {
  const { t } = useLingui()
  const open = useConsole((s) => s.helping)
  const setHelping = useConsole((s) => s.setHelping)
  const profile = useConsole((s) => s.shown.profile)
  const profileLoader = useProfileLoader(profile)
  const loaderId = profileLoader?.id ?? ''
  const paste = profileLoader?.paste ?? ''
  const pasteHost = paste ? new URL(paste).host : ''
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
    shareLogText(viewingRun, paste !== '', {
      live: () => Log(game, profile),
      run: (id) => RunLog(game, profile, id),
      runs: () => Runs(game, profile),
    }).then(
      (text) => {
        if (live) {
          setLog(text)
        }
      },
      (e: unknown) => {
        if (!live) {
          return
        }
        reportError(t`Could not read the game log`)(e)
        setHelping(false)
      },
    )
    return () => {
      live = false
    }
  }, [open, game, profile, viewingRun, paste, setHelping, t])

  const copy = (text: string) => copyText(text, t`Link copied`)

  const copyAndOpen = () => {
    if (!log) {
      return
    }
    copyText(hideUserName ? anonymize(log) : log, t`Log copied`, () => {
      close()
      return openPage(paste)
    })
  }

  const upload = () => {
    const token = uploadGen.current
    if (log === null || uploading) {
      return
    }
    setUploading(true)
    Upload(game, loaderId, hideUserName ? anonymize(log) : log)
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
    body = t`No log for this profile yet. Play it once, then try again.`
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
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: space.pad }}>
        {body ? (
          <>
            {log === null ? <LoadingRow>{body}</LoadingRow> : null}
            {log === null ? null : <Typography sx={{ fontSize: 14 }}>{body}</Typography>}
          </>
        ) : (
          <HelpLog
            log={log ?? ''}
            link={link}
            paste={paste}
            hideUserName={hideUserName}
            setHideUserName={setHideUserName}
            copy={copy}
          />
        )}
      </DialogContent>
      <DialogActions sx={{ px: space.pad, pb: space.pad }}>
        <Button disabled={uploading} onClick={close} sx={button}>
          {link || !log ? t`Close` : t`Cancel`}
        </Button>
        {log && paste ? (
          <Button variant="contained" onClick={copyAndOpen} sx={button}>
            {t`Copy log and open ${pasteHost}`}
          </Button>
        ) : null}
        {log && !link && !paste ? (
          <Button variant="contained" disabled={uploading} onClick={upload} sx={button}>
            {uploading ? t`Uploading…` : t`Upload log`}
          </Button>
        ) : null}
      </DialogActions>
    </Dialog>
  )
}
