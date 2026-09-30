import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'

const PERCENT = 100
const KB = 1024
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

export const isActive = (i: Item) => i.state === 'downloading' || i.state === 'installing'

// What the user still waits for: everything that neither finished nor was dropped, failures included.
export const isLeft = (i: Item) => !['done', 'skipped', 'cancelled'].includes(i.state)

// A file that is queued or under way for this mod in this profile; a failed one does not count, so it can be added again.
export const isPending = (i: Item) => isLeft(i) && i.state !== 'failed'

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

// "12 MB", or "0.4 MB" for a small file, from a size in KB.
export function megabytes(kb: number): string {
  const mb = kb / KB
  return mb >= 10 ? String(Math.round(mb)) : mb.toFixed(1)
}

export const megabytesPerSecond = (bytes: number) => (bytes / KB / KB).toFixed(1)

export const downloadedKb = (i: Item) => (i.sizeKb * i.progress) / PERCENT

// The wall-clock time of a Unix timestamp, as "14:05".
export const clockTime = (unix: number) =>
  new Date(unix * MS_PER_SECOND).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
