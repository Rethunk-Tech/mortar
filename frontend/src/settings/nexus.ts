import { create } from 'zustand'
import type { Account } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/models.ts'
import { Account as fetchAccount } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import { follow } from '../shell/follow.ts'

const signedOut: Pick<Account, 'signedIn' | 'name' | 'premium'> = {
  signedIn: false,
  name: '',
  premium: false,
}

export const useNexus = create<Pick<Account, 'signedIn' | 'name' | 'premium'>>(() => signedOut)

export const initNexus = () =>
  follow('nexus:changed', fetchAccount, (next) =>
    useNexus.setState({ signedIn: next.signedIn, name: next.name, premium: next.premium }),
  )
