import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type { Account } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/models.ts'
import { Account as fetchAccount } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'

const signedOut: Pick<Account, 'signedIn' | 'name' | 'premium'> = {
  signedIn: false,
  name: '',
  premium: false,
}

export const useNexus = create<Pick<Account, 'signedIn' | 'name' | 'premium'>>(() => signedOut)

export async function initNexus(): Promise<void> {
  const apply = (next: Account) =>
    useNexus.setState({ signedIn: next.signedIn, name: next.name, premium: next.premium })
  Events.On('nexus:changed', (event) => {
    apply(event.data)
  })
  apply(await fetchAccount())
}
