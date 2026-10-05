import { i18n } from '@lingui/core'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'

const PERCENT = 100
const MS_PER_SECOND = 1000

interface Totals {
  done: number
  active: number
  failed: number
  left: number
  sizeKb: number
  // Shares of the whole, in percent, for the bar split by state.
  doneShare: number
  activeShare: number
  failedShare: number
}

const isDownloading = (i: Item) => i.state === 'downloading'

const isWaiting = (i: Item) => i.state === 'queued'

const isLeft = (i: Item) => !['done', 'skipped', 'cancelled'].includes(i.state)

// A file that is queued or under way for this mod in this profile; a failed one does not count, so it can be added again.
const isPending = (i: Item) => isLeft(i) && i.state !== 'failed'

export const isActive = (i: Item) => i.state === 'downloading' || i.state === 'installing'

export function parallelDownloads(items: Item[]) {
  const counted = items.filter((i) => i.state !== 'skipped' && i.state !== 'cancelled')
  return {
    downloading: counted.filter(isDownloading).length,
    waiting: counted.filter(isWaiting).length,
  }
}

// What the user still waits for: everything that neither finished nor was dropped, failures included.
export const isClearableFinished = (state: string) =>
  state === 'done' || state === 'skipped' || state === 'cancelled'

// A GitHub mod is named by its repo, with a modId of 0; a Nexus mod by its modId and an empty repo.
export const pendingFor = (items: Item[], profileId: string, modId: number, repo = '') =>
  items.some(
    (i) => isPending(i) && i.profileId === profileId && i.modId === modId && i.repo === repo,
  )

export function totals(items: Item[]): Totals {
  const counted = items.filter((i) => i.state !== 'skipped' && i.state !== 'cancelled')
  const done = counted.filter((i) => i.state === 'done').length
  const active = counted.filter(isActive).length
  const failed = counted.filter((i) => i.state === 'failed').length
  const left = counted.filter(isLeft)
  const share = (n: number) => (counted.length === 0 ? 0 : (n * PERCENT) / counted.length)
  return {
    done,
    active,
    failed,
    left: left.length,
    sizeKb: left.reduce((sum, i) => sum + i.sizeKb, 0),
    doneShare: share(done),
    activeShare: share(active),
    failedShare: share(failed),
  }
}

export const downloadedKb = (i: Item) => (i.sizeKb * i.progress) / PERCENT

// The wall-clock time of a Unix timestamp, as "14:05".
export const clockTime = (unix: number) =>
  new Date(unix * MS_PER_SECOND).toLocaleTimeString(i18n.locale, {
    hour: '2-digit',
    minute: '2-digit',
  })

// The name of the profile an item installs into: '' when the open game's list no longer has it, null when the
// list loaded is another game's and cannot tell.
export function profileOf(
  item: Pick<Item, 'game' | 'profileId'>,
  game: string | undefined,
  profiles: readonly { id: string; name: string }[],
): string | null {
  if (item.game !== game) {
    return null
  }
  return profiles.find((p) => p.id === item.profileId)?.name ?? ''
}

// The letter-tile identity of a queue item.
export const tile = (i: Item) => ({
  id: i.repo || String(i.modId),
  name: i.name || i.repo || String(i.modId),
  picture: i.picture ?? '',
})
