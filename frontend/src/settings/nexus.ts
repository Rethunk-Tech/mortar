import { create } from 'zustand'
import type { Account } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/models.ts'
import { Account as fetchAccount } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import { follow } from '../shell/follow.ts'

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

export const initNexus = () =>
  follow('nexus:changed', fetchAccount, (next) =>
    useNexus.setState(applyNexusAccount(next, useNexus.getState().limits)),
  )
