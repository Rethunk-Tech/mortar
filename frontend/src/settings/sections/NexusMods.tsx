import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, Chip, TextField } from '@mui/material'
import { Events } from '@wailsio/runtime'
import { Check, LogIn, LogOut } from 'lucide-react'
import { type SubmitEvent, useEffect, useId, useState } from 'react'
import {
  APIKeyAvailable,
  SignIn,
  SignOut,
  TrackedCount,
  UntrackAll,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import {
  Get as GetSettings,
  SetAskEndorseMods,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { useProfiles } from '../../profiles/store.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { space } from '../../theme/density.ts'
import { errorKind } from '../../toasts/errorKind.ts'
import { type InlineError, inlineError, reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { usePending } from '../../toasts/usePending.ts'
import { useNexus } from '../nexus.ts'
import { PrefSwitch } from '../PrefControls.tsx'
import { PrefByKey } from '../PrefRow.tsx'
import { Searchable, SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { NexusSSO } from './NexusSSO.tsx'
import { useNxmHandler } from './nxmHandler.tsx'

function UntrackConfirmDialog({
  unused,
  busy,
  run,
  gameId,
  gameName,
  trackedCount,
  setTrackedCount,
  onCancel,
}: {
  unused: boolean | null
  busy: boolean
  run: ReturnType<typeof usePending>[1]
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
    run(
      async () => {
        const result = await UntrackAll(gameId, unused)
        const toast = {
          kind: result.stoppedForLimit ? 'warning' : 'success',
          title: plural(result.untracked, {
            one: 'Untracked # mod',
            other: 'Untracked # mods',
          }),
          ...(result.stoppedForLimit
            ? { body: t`Stopped at the API limit; ${result.remaining} left` }
            : {}),
        } as const
        pushToast(toast)
        setTrackedCount(result.remaining)
        onCancel()
      },
      { errorTitle: t`Could not untrack mods` },
    )
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
          : t`${plural(trackedCount, { one: `Untrack all # tracked ${gameName} mod on Nexus? Nexus has no undo for this.`, other: `Untrack all # tracked ${gameName} mods on Nexus? Nexus has no undo for this.` })}`
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
  const [busy, run] = usePending()
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
    trackedText = t`${plural(trackedCount, { one: `# ${gameName} mod tracked on Nexus`, other: `# ${gameName} mods tracked on Nexus` })}`
  } else if (gameName) {
    trackedText = t`Untrack every ${gameName} mod on Nexus`
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.pad }}>
      <Searchable terms={`${t`Nexus account`} Nexus ${t`Sign out`} ${name}`}>
        <Alert
          variant="compact"
          severity="success"
          icon={<Check size={16} aria-hidden={true} />}
          action={
            <Button
              color="inherit"
              size="small"
              startIcon={<LogOut size={16} />}
              onClick={() => {
                SignOut()
                  .then(() =>
                    useToasts.getState().push({ kind: 'success', title: t`Signed out of Nexus` }),
                  )
                  .catch(reportUnexpected)
              }}
            >
              {t`Sign out`}
            </Button>
          }
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
        <SettingRow label={t`Untrack on Nexus`} description={trackedText}>
          <Box sx={{ display: 'flex', gap: space.gap }}>
            <Button
              variant="outlined"
              disabled={untrackOff}
              onClick={() => setConfirming(true)}
            >{t`Untrack unused…`}</Button>
            <Button
              variant="outlined"
              disabled={untrackOff}
              onClick={() => setConfirming(false)}
            >{t`Untrack all…`}</Button>
          </Box>
        </SettingRow>
      </SettingsSection>
      <SettingsSection title={t`Endorsements`}>
        <SettingRow label={t`Ask me to endorse mods I keep using`}>
          <PrefSwitch
            checked={askEndorse}
            onChange={onAskEndorse}
            label={t`Ask me to endorse mods I keep using`}
          />
        </SettingRow>
      </SettingsSection>
      <UntrackConfirmDialog
        unused={confirming}
        busy={busy}
        run={run}
        gameId={game?.id}
        gameName={gameName}
        trackedCount={trackedCount ?? 0}
        setTrackedCount={setTrackedCount}
        onCancel={() => setConfirming(null)}
      />
    </Box>
  )
}

/** The Nexus sign-in form and the nxm link offer that follows it; renders only the offer once signed in. */
export function NexusSignIn() {
  const { t } = useLingui()
  const keyId = useId()
  const signedIn = useNexus((s) => s.signedIn)
  const nxm = useNxmHandler()
  const [key, setKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [keyAllowed, setKeyAllowed] = useState(true)
  const [error, setError] = useState<InlineError | null>(null)
  useEffect(() => {
    APIKeyAvailable().then(setKeyAllowed).catch(reportUnexpected)
  }, [])
  const submit = (e: SubmitEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    SignIn(key)
      .then(() => {
        setKey('')
        nxm.offer()
      })
      .catch((err: unknown) =>
        setError(
          errorKind(err) === 'invalid'
            ? inlineError(err, t`${{ site: sourceLabel('nexus') }} rejected your API key`)
            : inlineError(err),
        ),
      )
      .finally(() => setBusy(false))
  }
  // A sign-in through the browser ends with an event, not a promise; the account update unmounts the form first.
  const { offer } = nxm
  useEffect(
    () =>
      Events.On('nexus:sso', (e) => {
        if (e.data.state === 'done') {
          offer()
        }
      }),
    [offer],
  )
  if (signedIn) {
    return nxm.dialog
  }
  return (
    <Searchable terms={`${t`Nexus account`} Nexus ${t`Personal API key`} ${t`Sign in`}`}>
      {nxm.dialog}
      <NexusSSO />
      {keyAllowed ? (
        <SettingRow
          label={t`Personal API key`}
          description={t`Kept in your system keyring. Find it on Nexus under Settings, API Keys.`}
        >
          <Box
            component="form"
            onSubmit={submit}
            sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
              <TextField
                id={keyId}
                slotProps={{ htmlInput: { 'aria-label': t`Personal API key` } }}
                type="password"
                size="small"
                autoComplete="off"
                value={key}
                onChange={(e) => setKey(e.target.value)}
                sx={{ width: 260, maxWidth: '100%' }}
              />
              <Button
                type="submit"
                variant="contained"
                startIcon={<LogIn size={16} />}
                disabled={busy || key.trim() === ''}
                sx={{ flexShrink: 0 }}
              >
                {t`Sign in`}
              </Button>
            </Box>
          </Box>
        </SettingRow>
      ) : null}
      {error ? (
        <Alert variant="compact" severity="error" title={error.details}>
          {error.message}
        </Alert>
      ) : null}
    </Searchable>
  )
}

export function NexusMods() {
  const signedIn = useNexus((s) => s.signedIn)
  const name = useNexus((s) => s.name)
  const premium = useNexus((s) => s.premium)
  const [askEndorse, setAskEndorse] = useState(false)
  useEffect(() => {
    GetSettings()
      .then((settings) => setAskEndorse(settings.askEndorseMods ?? false))
      .catch(reportUnexpected)
  }, [])
  const onAskEndorse = (on: boolean) => {
    setAskEndorse(on)
    SetAskEndorseMods(on).catch((err: unknown) => {
      setAskEndorse(!on)
      reportUnexpected(err)
    })
  }
  const { t } = useLingui()
  return (
    <SettingsSection title={t`Nexus Mods`} tiles={false}>
      <NexusSignIn />
      {signedIn ? (
        <NexusModsSignedIn
          name={name}
          premium={premium}
          askEndorse={askEndorse}
          onAskEndorse={onAskEndorse}
        />
      ) : null}
    </SettingsSection>
  )
}
