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
  Tooltip,
} from '@mui/material'
import { Check, LogIn, LogOut } from 'lucide-react'
import { type SubmitEvent, useEffect, useId, useState } from 'react'
import {
  SignIn,
  SignOut,
  TrackedCount,
  UntrackAll,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import {
  Get as GetSettings,
  SetAskEndorseMods,
  SetNexusPreferredDownloadServer,
  SetNxmRedirectOtherGames,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useProfiles } from '../../profiles/store.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { errorText, reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { NexusMeter } from '../NexusMeter.tsx'
import { useNexus } from '../nexus.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { NexusDownloadPrefs } from './NexusDownloadPrefs.tsx'
import { useNxmHandler } from './nxmHandler.tsx'
import { nxmOwnerName } from './nxmOwnerName.ts'

function UntrackConfirmDialog({
  unused,
  busy,
  setBusy,
  gameId,
  gameName,
  trackedCount,
  setTrackedCount,
  onCancel,
}: {
  unused: boolean | null
  busy: boolean
  setBusy: (v: boolean) => void
  gameId: string | undefined
  gameName: string
  trackedCount: number
  setTrackedCount: (n: number) => void
  onCancel: () => void
}) {
  const { t } = useLingui()
  const pushToast = useToasts((s) => s.push)
  const untrack = () => {
    if (unused === null || !gameId) {
      return
    }
    setBusy(true)
    UntrackAll(gameId, unused)
      .then((result) => {
        const toast = {
          kind: result.stoppedForLimit ? 'warning' : 'success',
          title: t`Untracked ${result.untracked} mods`,
          ...(result.stoppedForLimit
            ? { body: t`Stopped at the API limit; ${result.remaining} left` }
            : {}),
        } as const
        pushToast(toast)
        setTrackedCount(result.remaining)
        onCancel()
      })
      .catch((err: unknown) =>
        pushToast({ kind: 'error', title: errorText(err) ?? t`Could not untrack mods` }),
      )
      .finally(() => setBusy(false))
  }
  return (
    <ConfirmDialog
      open={unused !== null}
      color="error"
      busy={busy}
      title={t`Untrack mods?`}
      body={
        unused
          ? t`Untrack the ${gameName} mods none of your profiles use, out of ${trackedCount} tracked? Nexus has no undo for this.`
          : t`Untrack all ${trackedCount} tracked ${gameName} mods on Nexus? Nexus has no undo for this.`
      }
      confirmLabel={t`Untrack mods`}
      onCancel={onCancel}
      onConfirm={untrack}
    />
  )
}

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
  const [trackedCount, setTrackedCount] = useState<number | null>(null)
  const [confirming, setConfirming] = useState<boolean | null>(null)
  const [busy, setBusy] = useState(false)
  // The game last opened; untracking is per game, and Settings has no game of its own.
  const game = useProfiles((s) => s.game)

  useEffect(() => {
    if (game) {
      TrackedCount(game.id).then(setTrackedCount).catch(reportUnexpected)
    }
  }, [game])

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
      <SettingsSection title={t`Tracking`}>
        <Box sx={{ px: 2, pt: 1, fontSize: 13, color: 'text.secondary' }}>
          {trackedCount === null
            ? t`Tracked for Stardew Valley`
            : t`${trackedCount} tracked for Stardew Valley`}
        </Box>
        <SettingRow
          label={t`Untrack all…`}
          description={t`Untrack every Stardew Valley mod on Nexus`}
        >
          <Tooltip title={t`Untrack every Stardew Valley mod on Nexus`}>
            <span>
              <Button
                variant="outlined"
                disabled={busy || trackedCount === null || trackedCount === 0}
                onClick={() => setConfirming(false)}
              >{t`Untrack all…`}</Button>
            </span>
          </Tooltip>
        </SettingRow>
        <SettingRow
          label={t`Untrack unused…`}
          description={t`Untrack mods not used by any profile`}
        >
          <Tooltip title={t`Untrack mods not used by any profile`}>
            <span>
              <Button
                variant="outlined"
                disabled={busy || trackedCount === null || trackedCount === 0}
                onClick={() => setConfirming(true)}
              >{t`Untrack unused…`}</Button>
            </span>
          </Tooltip>
        </SettingRow>
      </SettingsSection>
      {premium ? null : (
        <Box sx={{ fontSize: 13, color: 'text.secondary' }}>
          {t`Free accounts need one click on Nexus for every download. Mortar opens each file's page in turn and takes the download from your click.`}
        </Box>
      )}
      <NexusMeter />
      <SettingsSection title={t`Endorsements`}>
        <SettingRow label={t`Ask me to endorse mods I keep using`}>
          <Switch checked={askEndorse} onChange={(_, on) => onAskEndorse(on)} />
        </SettingRow>
      </SettingsSection>
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
      <SettingsSection
        title={t`Downloads`}
        prefKeys={['autoTrackNexus', 'parallelDownloads', 'nxmDefaultProfile']}
      >
        <SettingRow
          label={t`Handle "Mod Manager Download" links`}
          description={t`Clicking these links on Nexus starts the download in Mortar.`}
        >
          <Tooltip
            title={owner ? t`${owner} opens these links now.` : t`Mortar handles these links now.`}
          >
            <Switch checked={nxm.handled} onChange={(_, on) => nxm.toggle(on)} />
          </Tooltip>
        </SettingRow>
        <NexusDownloadPrefs />
      </SettingsSection>
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
      <UntrackConfirmDialog
        unused={confirming}
        busy={busy}
        setBusy={setBusy}
        gameId={game?.id}
        gameName={game?.name ?? ''}
        trackedCount={trackedCount ?? 0}
        setTrackedCount={setTrackedCount}
        onCancel={() => setConfirming(null)}
      />
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
