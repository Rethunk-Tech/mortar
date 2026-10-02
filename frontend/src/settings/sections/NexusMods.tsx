import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  Box,
  Button,
  Chip,
  FormControl,
  FormControlLabel,
  InputLabel,
  MenuItem,
  Select,
  Switch,
  TextField,
} from '@mui/material'
import { Check, LogIn, LogOut } from 'lucide-react'
import { type SubmitEvent, useId, useState } from 'react'
import {
  SignIn,
  SignOut,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import {
  SetNexusPreferredDownloadServer,
  SetNxmRedirectOtherGames,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { errorText, reportUnexpected } from '../../toasts/report.ts'
import { NexusMeter } from '../NexusMeter.tsx'
import { useNexus } from '../nexus.ts'
import { useSettings } from '../store.ts'
import { useNxmHandler } from './nxmHandler.tsx'
import { nxmOwnerName } from './nxmOwnerName.ts'

export function NexusMods() {
  const { t } = useLingui()

  const keyId = useId()
  const serverId = useId()
  const { signedIn, name, premium } = useNexus()
  const nxm = useNxmHandler()
  const preferredServer = useSettings((s) => s.nexusPreferredDownloadServer)
  const seenServers = useSettings((s) => s.nexusSeenDownloadServers)
  const nxmPrevious = useSettings((s) => s.nxmPrevious)
  const nxmPreviousName = useSettings((s) => s.nxmPreviousName)
  const redirectOther = useSettings((s) => s.nxmRedirectOtherGames ?? nxmPrevious !== '')
  const owner = nxmOwnerName(nxm.handled, nxm.owner)
  const redirectName = nxmOwnerName(true, nxmPreviousName || nxmPrevious)
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
        <NexusMeter />
        {seenServers && seenServers.length > 0 ? (
          <FormControl size="small" fullWidth={true}>
            {/* Automatic is the empty value, so the label must stay raised and the field must show it. */}
            <InputLabel id={serverId} shrink={true}>
              {t`Preferred download server`}
            </InputLabel>
            <Select
              labelId={serverId}
              label={t`Preferred download server`}
              value={preferredServer}
              displayEmpty={true}
              notched={true}
              renderValue={(v) => (v === '' ? t`Automatic` : String(v))}
              onChange={(e) => {
                SetNexusPreferredDownloadServer(String(e.target.value)).catch(reportUnexpected)
              }}
            >
              <MenuItem value="">{t`Automatic`}</MenuItem>
              {seenServers?.map((server) => (
                <MenuItem key={server} value={server}>
                  {server}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        ) : null}
        <FormControlLabel
          control={<Switch checked={nxm.handled} onChange={(_, on) => nxm.toggle(on)} />}
          label={t`Handle Nexus "Mod Manager Download" links`}
        />
        {owner ? (
          <Box sx={{ fontSize: 13, color: 'rgba(225,225,230,0.95)' }}>
            {t`${owner} opens these links now.`}
          </Box>
        ) : null}
        {nxm.handled && nxmPrevious ? (
          <FormControlLabel
            control={
              <Switch
                checked={redirectOther}
                onChange={(_, on) => SetNxmRedirectOtherGames(on).catch(reportUnexpected)}
              />
            }
            label={t`Send other games' links to ${redirectName}`}
          />
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
