import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip } from '@mui/material'
import { type ReactNode, useEffect, useState } from 'react'
import { Retry } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/service.ts'
import { GitHubRate } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { formatWhen } from '../../i18n/formatWhen.ts'
import { useOffline } from '../../shell/offline.ts'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { usePending } from '../../toasts/usePending.ts'
import { NexusMeter } from '../NexusMeter.tsx'
import { useNexus } from '../nexus.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettingsRelevant } from '../useSettingsSearch.ts'
import { AccountRow, type AccountTone } from './AccountRow.tsx'

// Go's zero time.
const NEVER_YEAR = 2000
const seen = (iso: string) => new Date(iso).getFullYear() >= NEVER_YEAR

// Reads its own state: the network poll landing re-renders this row, not the block's account and quota rows.
function useConnection(id: string) {
  const { t } = useLingui()
  const state = useOffline((s) => s.states.find((x) => x.id === id))
  const [pending, run] = usePending()
  let chip = <Chip size="small" label={t`Not contacted yet`} />
  let tone: AccountTone = 'none'
  let text: string = t`Mortar has not asked ${{ source: sourceLabel(id) }} this session.`
  let detail: ReactNode = text
  if (state?.unreachable) {
    chip = <Chip size="small" color="error" label={t`Unreachable`} />
    tone = 'bad'
    const reasons: Record<string, string> = {
      dns: t`its address could not be looked up`,
      timeout: t`it took too long to answer`,
      refused: t`the connection was refused`,
    }
    const reason = reasons[state.lastReason] ?? t`the connection failed`
    text = seen(state.lastFail)
      ? t`No answer ${{ when: formatWhen(state.lastFail) }}: ${reason}`
      : t`Could not be reached: ${reason}`
    // The raw error is for a bug report, so it waits in the tooltip.
    detail = <span title={state.lastError}>{text}</span>
  } else if (state && seen(state.lastOK)) {
    chip = <Chip size="small" color="success" label={t`Reachable`} />
    tone = 'ok'
    text = t`Last answered ${{ when: formatWhen(state.lastOK) }}`
    detail = text
  }
  const recheck = (
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
  )
  return { chip, tone, detail, text, recheck }
}

/** A service with no account: its connection status stands where the account line would. */
export function ConnectionAccountRow({ source }: { source: string }) {
  const { tone, text, recheck } = useConnection(source)
  return <AccountRow source={source} status={text} tone={tone} action={recheck} />
}

export function Connection({ source }: { source: string }) {
  const { t } = useLingui()
  const { chip, detail, recheck } = useConnection(source)
  return (
    <SettingRow label={t`Connection`} description={detail}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        {chip}
        {recheck}
      </Box>
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
        // A reset already past means the count shown is stale; the next request refreshes it.
        new Date(rate.reset).getTime() > Date.now()
          ? t`resets ${{ when: formatWhen(rate.reset) }}`
          : '',
      ]
        .filter(Boolean)
        .join(' · ')
    : unknown
  return (
    <SettingRow label={t`Requests`} description={detail}>
      {null}
    </SettingRow>
  )
}

export function Quota({ source }: { source: string }) {
  const { t } = useLingui()
  const premium = useNexus((s) => s.premium)
  if (source === 'github') {
    return <GitHubQuota />
  }
  if (source !== 'nexus') {
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
