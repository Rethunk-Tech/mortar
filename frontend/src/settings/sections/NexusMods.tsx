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
import { type SubmitEvent, useEffect, useId, useState } from 'react'
import {
  SignIn,
  SignOut,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import {
  Get as GetSettings,
  SetAskEndorseMods,
  SetNexusPreferredDownloadServer,
  SetNxmRedirectOtherGames,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { errorText, reportUnexpected } from '../../toasts/report.ts'
import { NexusMeter } from '../NexusMeter.tsx'
import { useNexus } from '../nexus.ts'
import { useSettings } from '../store.ts'
import { useNxmHandler } from './nxmHandler.tsx'
import { nxmOwnerName } from './nxmOwnerName.ts'

function NexusModsSignedIn({
  serverId,
  name,
  premium,
  preferredServer,
  seenServers,
  nxm,
  owner,
  nxmPrevious,
  redirectOther,
  redirectName,
  askEndorse,
  onAskEndorse,
}: {
  serverId: string
  name: string
  premium: boolean
  preferredServer: string
  seenServers: string[] | null
  nxm: ReturnType<typeof useNxmHandler>
  owner: string
  nxmPrevious: string
  redirectOther: boolean
  redirectName: string
  askEndorse: boolean
  onAskEndorse: (on: boolean) => void
}) {
  const { t } = useLingui()
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
      <FormControlLabel
        sx={{ m: 0, alignItems: 'flex-start' }}
        control={<Switch checked={askEndorse} onChange={(_, on) => onAskEndorse(on)} />}
        label={
          <Box component="span" sx={{ display: 'block', fontSize: 14 }}>
            {t`Ask me to endorse mods I keep using`}
          </Box>
        }
      />
      {seenServers && seenServers.length > 0 ? (
        <FormControl size="small" sx={{ maxWidth: 360 }}>
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
        sx={{ m: 0, alignItems: 'flex-start' }}
        control={<Switch checked={nxm.handled} onChange={(_, on) => nxm.toggle(on)} />}
        label={
          <Box>
            <Box component="span" sx={{ display: 'block', fontSize: 14 }}>
              {t`Handle Nexus "Mod Manager Download" links`}
            </Box>
            <Box
              component="span"
              sx={{ display: 'block', fontSize: 13, color: 'rgba(225,225,230,0.95)' }}
            >
              {owner
                ? t`${owner} opens these links now. Clicking Mod Manager Download on Nexus then starts the download in Mortar. Turning this off gives the links back to the app that had them.`
                : t`Clicking Mod Manager Download on Nexus then starts the download in Mortar. Turning this off gives the links back to the app that had them.`}
            </Box>
          </Box>
        }
      />
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
      {nxm.dialog}
    </Box>
  )
}

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
  const [askEndorse, setAskEndorse] = useState(true)
  useEffect(() => {
    GetSettings()
      .then((settings) => setAskEndorse(settings.askEndorseMods ?? true))
      .catch(reportUnexpected)
  }, [])
  const onAskEndorse = (on: boolean) => {
    setAskEndorse(on)
    SetAskEndorseMods(on).catch((err: unknown) => {
      setAskEndorse(!on)
      reportUnexpected(err)
    })
  }
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
      <NexusModsSignedIn
        serverId={serverId}
        name={name}
        premium={premium}
        preferredServer={preferredServer}
        seenServers={seenServers}
        nxm={nxm}
        owner={owner}
        nxmPrevious={nxmPrevious}
        redirectOther={redirectOther}
        redirectName={redirectName}
        askEndorse={askEndorse}
        onAskEndorse={onAskEndorse}
      />
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
      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
        <TextField
          id={keyId}
          type="password"
          size="small"
          autoComplete="off"
          value={key}
          onChange={(e) => setKey(e.target.value)}
          helperText={t`Kept in your system keyring. Find it on Nexus under Settings, API Keys.`}
          sx={{ width: 420, maxWidth: '100%' }}
        />
        <Button
          type="submit"
          variant="contained"
          startIcon={<LogIn size={16} />}
          disabled={busy || key.trim() === ''}
          sx={{ whiteSpace: 'nowrap', flexShrink: 0, height: 40 }}
        >
          {t`Sign in`}
        </Button>
      </Box>
      {error ? <Alert severity="error">{error}</Alert> : null}
    </Box>
  )
}
