import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'

type CardState =
  | { kind: 'idle' }
  | { kind: 'waiting-nexus' }
  | { kind: 'queued' }
  | { kind: 'downloading'; percent: number }
  | { kind: 'installing' }
  | { kind: 'attention' }
  | { kind: 'done' }
  | { kind: 'failed'; itemId: string }

const IDLE: CardState = { kind: 'idle' }

function isCardItem(item: Item, source: string, id: string): boolean {
  switch (source) {
    case 'nexus':
      return item.modId === Number(id) && item.repo === '' && !item.package
    case 'github':
      return item.repo === id
    case 'thunderstore':
      return !item.source && item.package?.toLowerCase() === id.toLowerCase()
    default:
      return item.source === source && item.package === id
  }
}

function stateOf(item: Item): CardState {
  switch (item.state) {
    case 'waiting-click':
      return { kind: 'waiting-nexus' }
    case 'queued':
      return { kind: 'queued' }
    case 'downloading':
      return { kind: 'downloading', percent: Math.round(item.progress) }
    case 'installing':
      return { kind: 'installing' }
    case 'done':
      return { kind: 'done' }
    case 'failed':
      return { kind: 'failed', itemId: item.id }
    case 'skipped':
    case 'cancelled':
      return IDLE
    default:
      return { kind: 'attention' }
  }
}

// What a card shows for its mod on the source it installs from: the newest queue item for that mod in the profile.
// Dependencies are their own mods, so the card tracks only the item it asked for.
function cardState(items: Item[], source: string, id: string, profileID: string): CardState {
  const mine = items.filter(
    (i) => i.profileId === profileID && i.kind !== 'dependency' && isCardItem(i, source, id),
  )
  const last = mine.at(-1)
  return last ? stateOf(last) : IDLE
}

const ACTIVE = new Set<CardState['kind']>([
  'waiting-nexus',
  'queued',
  'downloading',
  'installing',
  'attention',
])

// What the card shows once the profile's own word is weighed against the queue's: an active item outranks the
// profile (the mod is on its way), the profile outranks finished items (a removed mod is not "Installed" and a mod
// installed another way is not "Failed"), and a done item counts only when this card watched it finish, since the
// search result that says "installed" is older than that install.
function shownState(live: CardState, installed: boolean, watched: boolean): CardState {
  if (ACTIVE.has(live.kind)) {
    return live
  }
  if (installed) {
    return IDLE
  }
  if (live.kind === 'failed' || (live.kind === 'done' && watched)) {
    return live
  }
  return IDLE
}

function isActive(state: CardState): boolean {
  return ACTIVE.has(state.kind)
}

// Whether the profile's entries hold the mod, by the entry's own source: Nexus by mod id, GitHub by repository, any
// other source by the package name it was installed under (the same rule as the backend's browse.Holdings).
function inProfile(profile: Profile | undefined, source: string, id: string): boolean {
  const want = id.toLowerCase()
  return (profile?.entries ?? []).some((e) => {
    const src = e.source
    if (src.kind !== source) {
      return false
    }
    if (source === 'nexus') {
      return String(src.modId) === id
    }
    return ((source === 'github' ? src.repo : src.name) ?? '').toLowerCase() === want
  })
}

// Whether the card counts its mod as installed. The profile's entries are the live word and the search's flag counts
// only until the profile changes, except for a loader: no entry holds it, so its flag stands until the next search.
function isInstalled(held: boolean, searched: boolean, loader: boolean, fresh: boolean): boolean {
  return held || (searched && (loader || fresh))
}

export type { CardState }
export { cardState, inProfile, isActive, isInstalled, shownState }
