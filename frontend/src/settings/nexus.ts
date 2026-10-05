import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type { Account } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import { Account as fetchAccount } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { i18n } from '../i18n/index.ts'
import { follow } from '../shell/follow.ts'
import { useToasts } from '../toasts/store.ts'

const emptyLimits = {
  known: false,
  daily: { remaining: 0, limit: 0, reset: '' },
  hourly: { remaining: 0, limit: 0, reset: '' },
}

type Limits = Account['limits']

const signedOut: Pick<Account, 'signedIn' | 'name' | 'premium' | 'limits'> = {
  signedIn: false,
  name: '',
  premium: false,
  limits: emptyLimits,
}

const asLimits = (raw: Limits | undefined, fallback: Limits): Limits => {
  if (!raw) {
    return fallback
  }
  return {
    known: raw.known === true,
    daily: raw.daily ?? emptyLimits.daily,
    hourly: raw.hourly ?? emptyLimits.hourly,
  }
}

export const applyNexusAccount = (
  next: Pick<Account, 'signedIn' | 'name' | 'premium' | 'limits'>,
  current: Limits = emptyLimits,
) => {
  const limits = asLimits(next.limits, emptyLimits)
  return {
    signedIn: next.signedIn,
    name: next.name,
    premium: next.premium,
    limits: current.known && !limits.known ? current : limits,
  }
}

export const useNexus = create<Pick<Account, 'signedIn' | 'name' | 'premium' | 'limits'>>(
  () => signedOut,
)

export const getInitialState = () => signedOut

/** Names the window under a tenth of its limit, with its reset so each window warns once; null while both have room. */
export const lowQuota = (limits: Limits): { window: 'daily' | 'hourly'; key: string } | null => {
  if (!limits.known) {
    return null
  }
  for (const window of ['hourly', 'daily'] as const) {
    const { remaining, limit, reset } = limits[window]
    if (limit > 0 && remaining < limit / 10) {
      return { window, key: `${window}:${reset}` }
    }
  }
  return null
}

export const initNexus = () => {
  let warned = ''
  return follow('nexus:changed', fetchAccount, (next) => {
    useNexus.setState(applyNexusAccount(next, useNexus.getState().limits))
    const limits = useNexus.getState().limits
    const low = lowQuota(limits)
    if (!low || low.key === warned) {
      return
    }
    warned = low.key
    const { remaining, limit, reset } = limits[low.window]
    const when = reset ? formatWhen(reset) : ''
    useToasts.getState().push({
      kind: 'warning',
      title: i18n._(msg`Nexus API requests are running low`),
      body:
        low.window === 'hourly'
          ? i18n._(msg`${remaining} of ${limit} left this hour; the count resets ${when}.`)
          : i18n._(msg`${remaining} of ${limit} left today; the count resets ${when}.`),
    })
  })
}
