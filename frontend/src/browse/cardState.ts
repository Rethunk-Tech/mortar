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

export type { CardState }
export { cardState }
