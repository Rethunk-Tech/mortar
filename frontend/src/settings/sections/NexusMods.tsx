import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, Chip, Switch, TextField } from '@mui/material'
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
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useProfiles } from '../../profiles/store.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { errorText, reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { NexusMeter } from '../NexusMeter.tsx'
import { useNexus } from '../nexus.ts'
import { PrefByKey } from '../PrefRow.tsx'
import { Searchable, SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useNxmHandler } from './nxmHandler.tsx'

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
  name,
  premium,
  askEndorse,
  onAskEndorse,
}: {
  name: string
  premium: boolean
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
  const gameName = game?.name ?? ''
  const untrackOff = busy || trackedCount === null || trackedCount === 0
  let trackedText = t`Untrack every mod you track on Nexus`
  if (gameName && trackedCount !== null) {
    trackedText = t`${trackedCount} ${gameName} mods tracked on Nexus`
  } else if (gameName) {
    trackedText = t`Untrack every ${gameName} mod on Nexus`
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Searchable terms={`${t`Nexus account`} Nexus ${t`Sign out`} ${name}`} loose={true}>
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
      </Searchable>
      <SettingsSection title={t`Tracking`}>
        <PrefByKey prefKey="autoTrackNexus" />
        <SettingRow label={t`Untrack all…`} description={trackedText}>
          <Button
            variant="outlined"
            disabled={untrackOff}
            onClick={() => setConfirming(false)}
          >{t`Untrack all…`}</Button>
        </SettingRow>
        <SettingRow
          label={t`Untrack unused…`}
          description={t`Untrack mods not used by any profile`}
        >
          <Button
            variant="outlined"
            disabled={untrackOff}
            onClick={() => setConfirming(true)}
          >{t`Untrack unused…`}</Button>
        </SettingRow>
      </SettingsSection>
      <SettingsSection title={t`Endorsements`}>
        <SettingRow label={t`Ask me to endorse mods I keep using`}>
          <Switch checked={askEndorse} onChange={(_, on) => onAskEndorse(on)} />
        </SettingRow>
      </SettingsSection>
      <SettingsSection
        title={t`API requests`}
        {...(premium
          ? {}
          : {
              description: t`Free accounts need one click on Nexus for every download. Mortar opens each file's page in turn and takes the download from your click.`,
            })}
      >
        <Searchable terms={`${t`API requests`} Nexus ${t`rate limit`}`}>
          <Box sx={{ px: 2, py: 1.5 }}>
            <NexusMeter />
          </Box>
        </Searchable>
      </SettingsSection>
      <UntrackConfirmDialog
        unused={confirming}
        busy={busy}
        setBusy={setBusy}
        gameId={game?.id}
        gameName={gameName}
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
  const { signedIn, name, premium } = useNexus()
  const nxm = useNxmHandler()
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
  const terms = `${t`Nexus account`} Nexus ${t`API key`} ${t`Sign in`}`
  if (signedIn) {
    return (
      <>
        <NexusModsSignedIn
          name={name}
          premium={premium}
          askEndorse={askEndorse}
          onAskEndorse={onAskEndorse}
        />
        {nxm.dialog}
      </>
    )
  }
  return (
    <Searchable terms={terms} loose={true}>
      {nxm.dialog}
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
    </Searchable>
  )
}
