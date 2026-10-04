import { NEXUS_DOMAIN } from './nexusUrl.ts'
export interface TrackedMod {
  modId: number
  domainName: string
}

export const isEndorsed = (status: string) => status === 'Endorsed'

export const isAbstained = (status: string) => status === 'Abstained'

export const isTracked = (mods: TrackedMod[] | undefined, modId: number) =>
  (mods ?? []).some(
    (m) => m.modId === modId && (m.domainName === '' || m.domainName === NEXUS_DOMAIN),
  )
