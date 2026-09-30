import { create } from 'zustand'
import type { Account } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/models.ts'
import { Account as fetchAccount } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import { follow } from '../shell/follow.ts'

const emptyLimits = {
  known: false,
  daily: { remaining: 0, limit: 0, reset: '' },
  hourly: { remaining: 0, limit: 0, reset: '' },
}

const signedOut: Pick<Account, 'signedIn' | 'name' | 'premium' | 'limits'> = {
  signedIn: false,
  name: '',
  premium: false,
  limits: emptyLimits,
}

export const useNexus = create<Pick<Account, 'signedIn' | 'name' | 'premium' | 'limits'>>(
  () => signedOut,
)

export const initNexus = () =>
  follow('nexus:changed', fetchAccount, (next) =>
    useNexus.setState({
      signedIn: next.signedIn,
      name: next.name,
      premium: next.premium,
      limits: next.limits ?? emptyLimits,
    }),
  )
