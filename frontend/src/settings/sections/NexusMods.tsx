import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, Chip, FormControlLabel, Switch, TextField } from '@mui/material'
import { Check, LogIn, LogOut } from 'lucide-react'
import { type SubmitEvent, useId, useState } from 'react'
import {
  SignIn,
  SignOut,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import { errorText, reportUnexpected } from '../../toasts/report.ts'
import { useNexus } from '../nexus.ts'
import { useNxmHandler } from './nxmHandler.tsx'
import { nxmOwnerName } from './nxmOwnerName.ts'

export function NexusMods() {
  const { t } = useLingui()
  const keyId = useId()
  const { signedIn, name, premium } = useNexus()
  const nxm = useNxmHandler()
  const owner = nxmOwnerName(nxm.handled, nxm.owner)
  const [key, setKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const submit = (e: SubmitEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    SignIn(key)
      .then(() => {
        setKey('')
        nxm.offer()
      })
      .catch((err: unknown) => setError(errorText(err) ?? t`Could not sign in`))
      .finally(() => setBusy(false))
  }
  if (signedIn) {
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        <Alert
          severity="success"
          icon={<Check size={16} aria-hidden={true} />}
          action={
            <Button
              color="inherit"
              size="small"
              startIcon={<LogOut size={16} />}
              onClick={() => {
                SignOut().catch(reportUnexpected)
              }}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {t`Sign out`}
            </Button>
          }
          sx={{ alignItems: 'center', fontSize: 14 }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <span>{t`Signed in as ${name}`}</span>
            <Chip
              size="small"
              color={premium ? 'primary' : 'default'}
              label={premium ? t`Premium` : t`Free`}
            />
          </Box>
        </Alert>
        {premium ? null : (
          <Box sx={{ fontSize: 14, lineHeight: 1.5 }}>
            {t`Free accounts need one click on Nexus for every download. Mortar opens each file's page in turn and takes the download from your click.`}
          </Box>
        )}
        <FormControlLabel
          control={<Switch checked={nxm.handled} onChange={(_, on) => nxm.toggle(on)} />}
          label={t`Handle Nexus "Mod Manager Download" links`}
        />
        {owner ? (
          <Box sx={{ fontSize: 13, color: 'rgba(225,225,230,0.95)' }}>
            {t`${owner} opens these links now.`}
          </Box>
        ) : null}
        <Box sx={{ fontSize: 13, color: 'rgba(225,225,230,0.95)' }}>
          {t`Clicking Mod Manager Download on Nexus then starts the download in Mortar. Turning this off gives the links back to the app that had them.`}
        </Box>
        {nxm.dialog}
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
