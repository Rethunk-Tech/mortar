import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, Chip, TextField } from '@mui/material'
import { Check, LogIn, LogOut } from 'lucide-react'
import { type SubmitEvent, useId, useState } from 'react'
import {
  SignIn,
  SignOut,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import { errorText, reportUnexpected } from '../../toasts/report.ts'
import { useNexus } from '../nexus.ts'

export function NexusMods() {
  const { t } = useLingui()
  const keyId = useId()
  const { signedIn, name, premium } = useNexus()
  const [key, setKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const submit = (e: SubmitEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    SignIn(key)
      .then(() => setKey(''))
      .catch((err: unknown) => setError(errorText(err) ?? t`Could not sign in`))
      .finally(() => setBusy(false))
  }
  if (signedIn) {
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            p: '12px 14px',
            bgcolor: 'rgba(12,223,100,0.12)',
            border: '1px solid rgba(12,223,100,0.4)',
            borderRadius: '6px',
            fontSize: 14,
          }}
        >
          <Check size={16} aria-hidden={true} />
          <span>{t`Signed in as ${name}`}</span>
          <Chip
            size="small"
            color={premium ? 'primary' : 'default'}
            label={premium ? t`Premium` : t`Free`}
          />
        </Box>
        {premium ? null : (
          <Box sx={{ fontSize: 14, lineHeight: 1.5 }}>
            {t`Free accounts need one click on Nexus for every download. Mortar opens each file's page in turn and takes the download from your click.`}
          </Box>
        )}
        <Button
          variant="outlined"
          startIcon={<LogOut size={16} />}
          onClick={() => {
            SignOut().catch(reportUnexpected)
          }}
          sx={{ alignSelf: 'flex-start', whiteSpace: 'nowrap' }}
        >
          {t`Sign out`}
        </Button>
      </Box>
    )
  }
  return (
    <Box
      component="form"
      onSubmit={submit}
      sx={{ display: 'flex', flexDirection: 'column', gap: '6px' }}
    >
      <Box component="label" htmlFor={keyId} sx={{ fontSize: 14, fontWeight: 600 }}>
        {t`Personal API key`}
      </Box>
      <Box sx={{ display: 'flex', gap: 1 }}>
        <TextField
          id={keyId}
          type="password"
          size="small"
          autoComplete="off"
          value={key}
          onChange={(e) => setKey(e.target.value)}
          sx={{ flexGrow: 1 }}
        />
        <Button
          type="submit"
          variant="contained"
          startIcon={<LogIn size={16} />}
          disabled={busy || key.trim() === ''}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Sign in`}
        </Button>
      </Box>
      <Box sx={{ fontSize: 13, color: 'rgba(225,225,230,0.95)' }}>
        {t`Kept in your system keyring. Find it on Nexus under Settings, API Keys.`}
      </Box>
      {error ? <Alert severity="error">{error}</Alert> : null}
    </Box>
  )
}
