import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { localId } from '../mods/dependents.ts'
import { useDetail } from '../mods/detail.ts'
import { sameId } from '../mods/lookup.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from './store.ts'

type Listener = () => void
const findAllListeners = new Set<Listener>()
let findAllPending = false

export interface ModHit {
  profileId: string
  profileName: string
  key: string
  id: string
  name: string
  version: string
  enabled: boolean
}

export function findModInProfiles(profiles: Profile[], query: string): ModHit[] {
  const q = query.trim().toLowerCase()
  if (!q) {
    return []
  }
  const out: ModHit[] = []
  for (const p of profiles) {
    for (const e of p.entries ?? []) {
      for (const m of e.mods ?? []) {
        if (m.name.toLowerCase().includes(q) || localId(m.id).toLowerCase().includes(q)) {
          out.push({
            profileId: p.id,
            profileName: p.name,
            key: e.key,
            id: m.id,
            name: m.name,
            version: m.version,
            enabled: !(e.disabled ?? []).some((id) => sameId(id, m.id)),
          })
        }
      }
    }
  }
  return out
}

export function openModInProfile(hit: Pick<ModHit, 'profileId' | 'key' | 'id'>): void {
  useDetail.getState().showAfterLoad({ key: hit.key, id: hit.id })
  useProfiles.getState().open(hit.profileId)
  useNav.getState().closeProfiles()
}

export function requestFindAllFocus() {
  findAllPending = true
  for (const fn of findAllListeners) {
    fn()
    findAllPending = false
  }
}

export function onFindAllFocus(fn: Listener): () => void {
  findAllListeners.add(fn)
  if (findAllPending) {
    fn()
    findAllPending = false
  }
  return () => {
    findAllListeners.delete(fn)
  }
}
