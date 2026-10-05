import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, Link } from '@mui/material'
import { type ReactNode, useEffect, useState } from 'react'
import type { State } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/models.ts'
import { Retry } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/service.ts'
import {
  GitHubLoggedIn,
  GitHubRate,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { Account as ItchAccountInfo } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/source/itch/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { formatWhen } from '../../i18n/formatWhen.ts'
import { openSettings } from '../../nav/store.ts'
import { refresh, useOffline } from '../../shell/offline.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { usePending } from '../../toasts/usePending.ts'
import { NexusMeter } from '../NexusMeter.tsx'
import { useNexus } from '../nexus.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'

const SOURCES = ['nexus', 'github', 'thunderstore', 'modrinth', 'itch'] as const
type SourceId = (typeof SOURCES)[number]

// Go's zero time.
const NEVER_YEAR = 2000
const seen = (iso: string) => new Date(iso).getFullYear() >= NEVER_YEAR

function Connection({ id, state }: { id: SourceId; state: State | undefined }) {
  const { t } = useLingui()
  const [pending, run] = usePending()
  let chip = <Chip size="small" label={t`Not contacted yet`} />
  let detail = t`Mortar has not asked ${sourceLabel(id)} this session.`
  if (state?.unreachable) {
    chip = <Chip size="small" color="error" label={t`Unreachable`} />
    const error = state.lastError
    detail = seen(state.lastFail)
      ? t`Failed ${formatWhen(state.lastFail)}: ${error}`
      : t`Could not be reached: ${error}`
  } else if (state && seen(state.lastOK)) {
    chip = <Chip size="small" color="success" label={t`Reachable`} />
    detail = t`Last answered ${formatWhen(state.lastOK)}`
  }
  return (
    <SettingRow label={t`Connection`} description={detail}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        {chip}
        <Button
          variant="outlined"
          size="small"
          disabled={pending}
          onClick={() =>
            run(() => Retry(id).then((next) => useOffline.getState().set(next)), {
              errorTitle: t`Could not recheck ${sourceLabel(id)}`,
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
  const nexus = useNexus()
  const { name } = nexus
  const [github, setGithub] = useState<boolean | null>(null)
  const [itch, setItch] = useState<string | null>(null)
  useEffect(() => {
    GitHubLoggedIn().then(setGithub).catch(reportUnexpected)
    ItchAccountInfo()
      .then((a) => setItch(a.signedIn ? a.name : ''))
      .catch(reportUnexpected)
  }, [])
  switch (id) {
    case 'nexus':
      return nexus.signedIn
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
          t`API key set for ${itch}`
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
  useEffect(() => {
    GitHubRate().then(setRate).catch(reportUnexpected)
  }, [])
  const detail = rate?.known
    ? t`${rate.remaining} of ${rate.limit} requests left · resets ${formatWhen(rate.reset)}`
    : t`Request counts appear after Mortar talks to GitHub.`
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
          : t`Free accounts click Download on Nexus for each file; Mortar opens each page in turn.`
      }
      block={true}
    >
      <NexusMeter />
    </SettingRow>
  )
}

export function Sources() {
  const states = useOffline((s) => s.states)
  useEffect(() => {
    refresh().catch(reportUnexpected)
  }, [])
  return (
    <>
      {SOURCES.map((id) => (
        <SettingsSection key={id} title={sourceLabel(id)}>
          <Connection id={id} state={states.find((s) => s.id === id)} />
          <Auth id={id} />
          <Quota id={id} />
        </SettingsSection>
      ))}
    </>
  )
}
