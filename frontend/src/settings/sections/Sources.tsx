import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, Link } from '@mui/material'
import { type ReactNode, useEffect, useState } from 'react'
import { Retry } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/service.ts'
import { GitHubRate } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { formatWhen } from '../../i18n/formatWhen.ts'
import { openSettings } from '../../nav/store.ts'
import { refresh, useOffline } from '../../shell/offline.ts'
import { githubLoggedIn, itchAccount } from '../../shell/sharedReads.ts'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { usePending } from '../../toasts/usePending.ts'
import { NexusMeter } from '../NexusMeter.tsx'
import { useNexus } from '../nexus.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettingsRelevant } from '../useSettingsSearch.ts'

const SOURCES = [
  'nexus',
  'github',
  'thunderstore',
  'modrinth',
  'curseforge',
  'itch',
  'vsmoddb',
  'paradox',
] as const
type SourceId = (typeof SOURCES)[number]

// Go's zero time.
const NEVER_YEAR = 2000
const seen = (iso: string) => new Date(iso).getFullYear() >= NEVER_YEAR

// Reads its own state: the network poll landing re-renders this row, not the section's account and quota rows.
function Connection({ id }: { id: SourceId }) {
  const { t } = useLingui()
  const state = useOffline((s) => s.states.find((x) => x.id === id))
  const [pending, run] = usePending()
  let chip = <Chip size="small" label={t`Not contacted yet`} />
  let detail: ReactNode = t`Mortar has not asked ${{ source: sourceLabel(id) }} this session.`
  if (state?.unreachable) {
    chip = <Chip size="small" color="error" label={t`Unreachable`} />
    const reasons: Record<string, string> = {
      dns: t`its address could not be looked up`,
      timeout: t`it took too long to answer`,
      refused: t`the connection was refused`,
    }
    const reason = reasons[state.lastReason] ?? t`the connection failed`
    // The raw error is for a bug report, so it waits in the tooltip.
    detail = (
      <span title={state.lastError}>
        {seen(state.lastFail)
          ? t`No answer ${{ when: formatWhen(state.lastFail) }}: ${reason}`
          : t`Could not be reached: ${reason}`}
      </span>
    )
  } else if (state && seen(state.lastOK)) {
    chip = <Chip size="small" color="success" label={t`Reachable`} />
    detail = t`Last answered ${{ when: formatWhen(state.lastOK) }}`
  }
  return (
    <SettingRow label={t`Connection`} description={detail}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        {chip}
        <Button
          variant="outlined"
          size="small"
          disabled={pending}
          onClick={() =>
            run(() => Retry(id).then((next) => useOffline.getState().set(next)), {
              errorTitle: t`Could not recheck ${{ source: sourceLabel(id) }}`,
            })
          }
        >
          {t`Recheck`}
        </Button>
      </Box>
    </SettingRow>
  )
}

function AccountLink({ children }: { children: ReactNode }) {
  return (
    <Link component="button" onClick={() => openSettings('accounts')} sx={{ fontSize: 'inherit' }}>
      {children}
    </Link>
  )
}

function useAuth(id: SourceId): { ok: boolean | null; text: ReactNode } {
  const { t } = useLingui()
  const nexusSignedIn = useNexus((s) => s.signedIn)
  const name = useNexus((s) => s.name)
  const [github, setGithub] = useState<boolean | null>(null)
  const [itch, setItch] = useState<string | null>(null)
  const relevant = useSettingsRelevant(
    t`Account`,
    t`Using your gh login`,
    t`No login, so the lower rate limit applies. Run gh auth login to raise it.`,
    t`API key set for ${{ key: '' }}`,
  )
  // Each source's row asks only about its own account.
  useEffect(() => {
    if (!relevant) {
      return
    }
    if (id === 'github') {
      githubLoggedIn().then(setGithub).catch(reportUnexpected)
    }
    if (id === 'itch') {
      itchAccount()
        .then((a) => setItch(a.signedIn ? a.name : ''))
        .catch(reportUnexpected)
    }
  }, [id, relevant])
  switch (id) {
    case 'nexus':
      return nexusSignedIn
        ? { ok: true, text: t`Signed in as ${name}` }
        : {
            ok: false,
            text: <AccountLink>{t`Not signed in. Add your API key in Accounts.`}</AccountLink>,
          }
    case 'github':
      return {
        ok: github,
        text: github
          ? t`Using your gh login`
          : t`No login, so the lower rate limit applies. Run gh auth login to raise it.`,
      }
    case 'itch':
      return {
        ok: itch === null ? null : itch !== '',
        text: itch ? (
          t`API key set for ${{ key: itch }}`
        ) : (
          <AccountLink>{t`API key not set. Itch.io search is off until you add one.`}</AccountLink>
        ),
      }
    default:
      return { ok: null, text: t`No account needed` }
  }
}

function Auth({ id }: { id: SourceId }) {
  const { t } = useLingui()
  const { ok, text } = useAuth(id)
  let chip = <Chip size="small" label={t`No account`} />
  if (ok === true) {
    chip = <Chip size="small" color="primary" label={t`Signed in`} />
  } else if (ok === false) {
    chip = <Chip size="small" color="warning" label={t`Not set up`} />
  }
  return (
    <SettingRow label={t`Account`} description={text}>
      {chip}
    </SettingRow>
  )
}

function GitHubQuota() {
  const { t } = useLingui()
  const [rate, setRate] = useState<Awaited<ReturnType<typeof GitHubRate>> | null>(null)
  const unknown = t`Request counts appear after Mortar talks to ${{ source: sourceLabel('github') }}.`
  const relevant = useSettingsRelevant(t`Requests`, unknown)
  useEffect(() => {
    if (relevant) {
      GitHubRate().then(setRate).catch(reportUnexpected)
    }
  }, [relevant])
  const detail = rate?.known
    ? [
        t`${{ remaining: rate.remaining }} of ${{ limit: rate.limit }} requests left`,
        t`resets ${{ when: formatWhen(rate.reset) }}`,
      ].join(' · ')
    : unknown
  return (
    <SettingRow label={t`Requests`} description={detail}>
      {null}
    </SettingRow>
  )
}

function Quota({ id }: { id: SourceId }) {
  const { t } = useLingui()
  const premium = useNexus((s) => s.premium)
  if (id === 'github') {
    return <GitHubQuota />
  }
  if (id !== 'nexus') {
    return null
  }
  return (
    <SettingRow
      label={t`Requests`}
      description={
        premium
          ? undefined
          : t`Free accounts click Download on ${{ source: sourceLabel('nexus') }} for each file; Mortar opens each page in turn.`
      }
      block={true}
    >
      <NexusMeter />
    </SettingRow>
  )
}

export function Sources() {
  const { t } = useLingui()
  const relevant = useSettingsRelevant(t`Connection`)
  useEffect(() => {
    if (relevant) {
      refresh().catch(reportUnexpected)
    }
  }, [relevant])
  return (
    <>
      {SOURCES.map((id) => (
        <SettingsSection key={id} title={sourceLabel(id)}>
          <Connection id={id} />
          <Auth id={id} />
          <Quota id={id} />
        </SettingsSection>
      ))}
    </>
  )
}
