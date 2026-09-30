import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, CircularProgress, Link, TextField } from '@mui/material'
import { Download, RefreshCw, RotateCcw } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { SetBackupsKept } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useNav } from '../../nav/store.ts'
import { errorMessage, errorText, reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { useSettings } from '../store.ts'
import { useMortarUpdate } from '../updates.ts'
import { lastOpenedGame } from './lastOpenedGame.ts'

const MIN_KEPT = 1
const MAX_KEPT = 50
const button = { whiteSpace: 'nowrap', alignSelf: 'flex-start' }

function MortarUpdate() {
  const { t } = useLingui()
  const { info, phase, release, error, load, check, install, restart } = useMortarUpdate()
  const [failed, setFailed] = useState('')
  const read = useCallback(() => {
    setFailed('')
    load().catch((e: unknown) => setFailed(errorMessage(e)))
  }, [load])
  useEffect(read, [read])
  if (!info) {
    return failed ? (
      <Alert
        severity="error"
        action={
          <Button color="inherit" size="small" onClick={read} sx={{ whiteSpace: 'nowrap' }}>
            {t`Retry`}
          </Button>
        }
        sx={{ fontSize: 14 }}
      >
        {t`Could not read this Mortar build's version: ${failed}`}
      </Alert>
    ) : null
  }
  if (info.off) {
    return (
      <Alert severity="info" sx={{ fontSize: 14 }}>
        {t`This is a development build of Mortar ${info.version}, so it does not check for updates.`}
      </Alert>
    )
  }
  const busy = phase === 'checking' || phase === 'installing' || phase === 'restarting'
  const states: Record<typeof phase, string> = {
    idle: t`Not checked yet.`,
    checking: t`Checking for updates…`,
    current: t`Mortar is up to date.`,
    available: t`Mortar ${release?.version ?? ''} is available.`,
    installing: t`Downloading and verifying Mortar ${release?.version ?? ''}…`,
    ready: t`Mortar ${release?.version ?? ''} is ready. Restart Mortar to finish updating.`,
    restarting: t`Restarting…`,
    error: t`Could not update: ${error}`,
  }
  let action = (
    <Button
      variant="outlined"
      startIcon={<RefreshCw size={16} />}
      disabled={busy}
      onClick={() => check().catch(reportUnexpected)}
      sx={button}
    >
      {t`Check now`}
    </Button>
  )
  if (phase === 'available') {
    action = (
      <Button
        variant="contained"
        startIcon={<Download size={16} />}
        onClick={() => install().catch(reportUnexpected)}
        sx={button}
      >
        {t`Download and install`}
      </Button>
    )
  } else if (phase === 'ready' || phase === 'restarting') {
    action = (
      <Button
        variant="contained"
        startIcon={<RotateCcw size={16} />}
        disabled={busy}
        onClick={() => restart().catch(reportUnexpected)}
        sx={button}
      >
        {t`Restart now`}
      </Button>
    )
  }
  return (
    <>
      <Box
        sx={{ fontSize: 14, color: 'rgba(225,225,230,0.95)' }}
      >{t`Installed: Mortar ${info.version}`}</Box>
      <Box
        role="status"
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          fontSize: 14,
          color: phase === 'error' ? 'error.main' : 'inherit',
        }}
      >
        {busy ? <CircularProgress size={14} /> : null}
        {states[phase]}
      </Box>
      {phase === 'available' && release?.notes ? (
        <Box
          sx={{
            fontSize: 13,
            whiteSpace: 'pre-wrap',
            maxHeight: 200,
            overflow: 'auto',
            p: 1.5,
            bgcolor: 'rgba(0,0,0,0.3)',
            borderRadius: '6px',
          }}
        >
          {release.notes}
        </Box>
      ) : null}
      {action}
    </>
  )
}

function BackupsKept() {
  const { t } = useLingui()
  const kept = useSettings((s) => s.backupsKept)
  const push = useToasts((s) => s.push)
  const [draft, setDraft] = useState(String(kept))
  useEffect(() => setDraft(String(kept)), [kept])
  const commit = () => {
    const n = Number(draft)
    if (!Number.isInteger(n) || n < MIN_KEPT || n > MAX_KEPT) {
      setDraft(String(kept))
      return
    }
    if (n !== kept) {
      SetBackupsKept(n).catch((err: unknown) => {
        const body = errorText(err)
        push({ kind: 'error', title: t`Couldn't save that setting`, ...(body ? { body } : {}) })
        setDraft(String(kept))
      })
    }
  }
  return (
    <TextField
      type="number"
      size="small"
      label={t`Backups kept`}
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === 'Enter' && e.target instanceof HTMLInputElement) {
          e.target.blur()
        }
      }}
      helperText={t`Saves are zipped before mods update; older backups beyond this many are deleted. ${MIN_KEPT} to ${MAX_KEPT}.`}
      slotProps={{ htmlInput: { min: MIN_KEPT, max: MAX_KEPT, step: 1 } }}
      sx={{ alignSelf: 'flex-start', width: 320, mt: 1 }}
    />
  )
}

export function Updates() {
  const { t } = useLingui()
  const game = lastOpenedGame(useSettings((s) => s.lastGame))
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Mortar`}</Box>
      <MortarUpdate />
      <Box sx={{ fontSize: 14, fontWeight: 600, pt: 1 }}>{t`Mods and SMAPI`}</Box>
      <Box sx={{ fontSize: 13, color: 'rgba(225,225,230,0.95)' }}>
        {t`Mod updates show on each profile's mod list, and SMAPI's in the game's settings.`}
      </Box>
      {game ? (
        <>
          <Link
            component="button"
            onClick={() => useNav.getState().openGame(game)}
            sx={{ alignSelf: 'flex-start', fontSize: 13, whiteSpace: 'nowrap' }}
          >
            {t`Review mod updates`}
          </Link>
          <Link
            component="button"
            onClick={() => useNav.setState({ route: { name: 'game-settings', game } })}
            sx={{ alignSelf: 'flex-start', fontSize: 13, whiteSpace: 'nowrap' }}
          >
            {t`Open Stardew Valley settings`}
          </Link>
        </>
      ) : null}
      <Box sx={{ fontSize: 14, fontWeight: 600, pt: 1 }}>{t`Save backups`}</Box>
      <BackupsKept />
    </Box>
  )
}
