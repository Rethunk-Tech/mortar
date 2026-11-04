import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { Events } from '@wailsio/runtime'
import { LogOut } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  SignOut,
  TrackedCount,
  UntrackAll,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { SetAskEndorseMods } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { useProfiles } from '../../profiles/store.ts'
import { ConfirmationDialog } from '../../shell/dialog/ConfirmationDialog.tsx'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { usePending } from '../../toasts/usePending.ts'
import { useNexus } from '../nexus.ts'
import { PrefSwitch } from '../PrefControls.tsx'
import { PrefByKey } from '../PrefRow.tsx'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { useSettingsRelevant } from '../useSettingsSearch.ts'
import { AccountRow, ServiceBlock } from './AccountRow.tsx'
import { NexusSSO } from './NexusSSO.tsx'
import { useNxmHandler } from './nxmHandler.tsx'
import { Connection, Quota } from './SourceConnection.tsx'

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
          title: plural(
            { count: result.untracked },
            {
              one: 'Untracked # mod',
              other: 'Untracked # mods',
            },
          ),
          ...(result.stoppedForLimit
            ? { body: t`Stopped at the API limit; ${{ remaining: result.remaining }} left` }
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
    <ConfirmationDialog
      open={unused !== null}
      tone="danger"
      busy={busy}
      title={t`Untrack mods?`}
      body={
        unused
          ? t`Untrack the ${{ game: gameName }} mods none of your profiles use, out of ${trackedCount} tracked? ${{ source: sourceLabel('nexus') }} has no undo for this.`
          : t`${plural({ count: trackedCount }, { one: `Untrack all # tracked ${{ game: gameName }} mod`, other: `Untrack all # tracked ${{ game: gameName }} mods` })} on ${{ source: sourceLabel('nexus') }}? ${{ source: sourceLabel('nexus') }} has no undo for this.`
      }
      confirmLabel={t`Untrack mods`}
      onCancel={onCancel}
      onConfirm={untrack}
    />
  )
}

function NexusModsSignedIn({
  askEndorse,
  onAskEndorse,
}: {
  askEndorse: boolean
  onAskEndorse: (on: boolean) => void
}) {
  const { t } = useLingui()
  const [confirming, setConfirming] = useState<boolean | null>(null)
  const [busy, run] = usePending()
  // The game last opened; untracking is per game, and Settings has no game of its own.
  const game = useProfiles((s) => s.game)

  const gameName = game?.name ?? ''
  const relevant = useSettingsRelevant(
    t`Untrack on ${{ source: sourceLabel('nexus') }}`,
    t`Untrack every ${{ game: gameName }} mod on ${{ source: sourceLabel('nexus') }}`,
  )
  const { data: trackedCount, setData: setTrackedCount } = useLoaded<number | null>(
    game && relevant ? () => TrackedCount(game.id) : null,
    [game, relevant],
    null,
    reportUnexpected,
  )
  const untrackOff = busy || trackedCount === null || trackedCount === 0
  let trackedText = t`Untrack every mod you track on ${{ source: sourceLabel('nexus') }}`
  if (gameName && trackedCount !== null) {
    trackedText = plural(
      { count: trackedCount },
      {
        one: `# ${{ game: gameName }} mod tracked on ${{ source: sourceLabel('nexus') }}`,
        other: `# ${{ game: gameName }} mods tracked on ${{ source: sourceLabel('nexus') }}`,
      },
    )
  } else if (gameName) {
    trackedText = t`Untrack every ${{ game: gameName }} mod on ${{ source: sourceLabel('nexus') }}`
  }

  return (
    <>
      <PrefByKey prefKey="autoTrackNexus" />
      <SettingRow
        label={t`Untrack on ${{ source: sourceLabel('nexus') }}`}
        description={trackedText}
      >
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
      <SettingRow label={t`Ask me to endorse mods I keep using`}>
        <PrefSwitch
          checked={askEndorse}
          onChange={onAskEndorse}
          label={t`Ask me to endorse mods I keep using`}
        />
      </SettingRow>
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
    </>
  )
}

/** The nxm link offer that follows a browser sign-in, which ends with an event rather than a promise. */
export function NexusSignIn() {
  const { t } = useLingui()
  const nxm = useNxmHandler(false)
  const { offer } = nxm
  useEffect(
    () =>
      Events.On('nexus:sso', (e) => {
        if (e.data.state === 'done') {
          useToasts
            .getState()
            .push({ kind: 'success', title: t`Signed in to ${{ source: sourceLabel('nexus') }}` })
          offer()
        }
      }),
    [offer, t],
  )
  return nxm.dialog
}

export function NexusMods() {
  const { t } = useLingui()
  const signedIn = useNexus((s) => s.signedIn)
  const name = useNexus((s) => s.name)
  const premium = useNexus((s) => s.premium)
  const askEndorse = useSettings((st) => st.askEndorseMods ?? false)
  const onAskEndorse = (on: boolean) => {
    SetAskEndorseMods(on).catch(reportUnexpected)
  }
  const signOut = () => {
    SignOut()
      .then(() =>
        useToasts.getState().push({
          kind: 'success',
          title: t`Signed out of ${{ source: sourceLabel('nexus') }}`,
        }),
      )
      .catch(reportUnexpected)
  }
  return (
    <SettingsSection>
      <ServiceBlock>
        <AccountRow
          source="nexus"
          tone={signedIn ? 'ok' : 'none'}
          status={
            signedIn
              ? `${name} · ${premium ? t`Premium account` : t`Free account`}`
              : t`Not signed in`
          }
          action={
            signedIn ? (
              <Button
                variant="outlined"
                size="small"
                startIcon={<LogOut size={16} />}
                onClick={signOut}
              >
                {t`Sign out`}
              </Button>
            ) : (
              <NexusSSO />
            )
          }
        />
        <NexusSignIn />
        {signedIn ? (
          <NexusModsSignedIn askEndorse={askEndorse} onAskEndorse={onAskEndorse} />
        ) : null}
        <Connection source="nexus" />
        <Quota source="nexus" />
      </ServiceBlock>
    </SettingsSection>
  )
}
