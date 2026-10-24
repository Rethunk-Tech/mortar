import { Events } from '@wailsio/runtime'
import { useEffect, useState } from 'react'
import { create } from 'zustand'
import type { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import {
  CachedDetails,
  MarkNexusSeen,
  MarkSeen,
  PrimeDetails,
  Details as readDetails,
  Seen,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { useNexus } from '../settings/nexus.ts'
import { readStates } from '../shell/offline.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { currentFiles, isNewer } from './nexusFormat.ts'
import { offersNexusDownload } from './nexusMark.ts'

interface Entry {
  details?: Details
  error?: string
}

// A Nexus mod id names a page only within its game's domain, so everything here is kept per game and id.
const nexusKey = (game: string, id: number) => `${game}:${id}`

const splitKey = (key: string): [string, number] => {
  const at = key.lastIndexOf(':')
  return [key.slice(0, at), Number(key.slice(at + 1))]
}

const put = (key: string, entry: Entry) =>
  useNexusDetails.setState((s) => ({ byKey: { ...s.byKey, [key]: entry } }))

// One read per mod per session: the backend refreshes a day-old cache on it, so a second read adds nothing. A failed
// read is forgotten so that signing in retries it, and keeps whatever the cache already gave.
const reads = new Map<string, Promise<void>>()

// One queue for every missing mod, so a large profile never bursts Nexus; the Nexus client also refuses locally once
// the rate-limit budget reaches its floor.
let line = Promise.resolve()
const queued = new Set<string>()
const waiting = new Set<string>()

const enqueue = (game: string, id: number) => {
  const key = nexusKey(game, id)
  if (queued.has(key) || reads.has(key)) {
    return
  }
  if (!useNexus.getState().signedIn) {
    waiting.add(key)
    return
  }
  waiting.delete(key)
  queued.add(key)
  line = line.then(async () => {
    queued.delete(key)
    if (useNexus.getState().signedIn) {
      await loadDetails(game, id)
    } else {
      waiting.add(key)
    }
  })
}

const useNexusDetails = create<{ byKey: Record<string, Entry | undefined> }>(() => ({
  byKey: {},
}))

// One mod's entry in a details snapshot.
const nexusEntryOf = (byKey: Record<string, Entry | undefined>, game: string, id: number) =>
  byKey[nexusKey(game, id)]

type NexusById = Record<number, Entry | undefined>

const views = new WeakMap<object, Map<string, NexusById>>()

// One game's entries by mod id, the same object for as long as the store holds the same entries, so a selector that
// returns it does not redraw on every read.
const detailsFor = (byKey: Record<string, Entry | undefined>, game: string): NexusById => {
  let perGame = views.get(byKey)
  if (!perGame) {
    perGame = new Map()
    views.set(byKey, perGame)
  }
  let view = perGame.get(game)
  if (!view) {
    view = {}
    const prefix = `${game}:`
    for (const [key, entry] of Object.entries(byKey)) {
      if (key.startsWith(prefix)) {
        view[Number(key.slice(prefix.length))] = entry
      }
    }
    perGame.set(game, view)
  }
  return view
}

// The game's entries by mod id, for a list that reads many mods.
const useGameNexusDetails = (game: string) => useNexusDetails((s) => detailsFor(s.byKey, game))

useNexus.subscribe((s, prev) => {
  if (s.signedIn && !prev.signedIn) {
    primeDetails(primed.game, primed.ids).catch(reportUnexpected)
    const keys = [...waiting]
    waiting.clear()
    for (const key of keys) {
      enqueue(...splitKey(key))
    }
    for (const [key, entry] of Object.entries(useNexusDetails.getState().byKey)) {
      if (!entry?.details) {
        reads.delete(key)
        enqueue(...splitKey(key))
      }
    }
  }
})

const loadDetails = (game: string, id: number) => {
  const key = nexusKey(game, id)
  let read = reads.get(key)
  if (!read) {
    read = readDetails(game, id).then(
      (details) => put(key, { details }),
      (e: unknown) => {
        reads.delete(key)
        put(key, { ...useNexusDetails.getState().byKey[key], error: errorMessage(e) })
      },
    )
    reads.set(key, read)
  }
  return read
}

let primed: { game: string; ids: number[] } = { game: '', ids: [] }
const primeAsked = new Set<string>()
const primeInFlight = new Set<string>()

// Stores the details that are new; when there are none the state object stays as it was, so subscribers do not wake.
const addDetails = (game: string, ids: number[], found: Record<string, Details | undefined>) =>
  useNexusDetails.setState((s) => {
    let next: Record<string, Entry | undefined> | undefined
    for (const id of ids) {
      const details = found[`${id}`]
      const key = nexusKey(game, id)
      if (details && !s.byKey[key]?.details) {
        next ??= { ...s.byKey }
        next[key] = { details }
      }
    }
    return next ? { byKey: next } : s
  })

// Shows what is cached for every mod at once, then fills the rest with the page data of all of them in a few batched
// requests. A mod's files and changelogs are read when it is opened, not for the whole list.
const primeDetails = async (game: string, ids: number[]) => {
  primed = { game, ids }
  await mergeCachedDetails(game, ids)
  if (!(game && useNexus.getState().signedIn)) {
    return
  }
  // A mod Nexus does not return (hidden, removed, a wrong id) stays unknown, so it is asked for once per session.
  const unknown = ids.filter((id) => {
    const key = nexusKey(game, id)
    return (
      id > 0 &&
      !useNexusDetails.getState().byKey[key]?.details &&
      !primeAsked.has(key) &&
      !primeInFlight.has(key)
    )
  })
  if (unknown.length === 0) {
    return
  }
  for (const id of unknown) {
    primeInFlight.add(nexusKey(game, id))
  }
  watchNexusReachable()
  // Only an answer settles an id: one the request could not fetch (offline, rate limited) is asked again, here or
  // when Nexus is reachable again. What did arrive is kept either way.
  const got = await PrimeDetails(game, unknown).catch(() => null)
  const found = got?.details ?? {}
  for (const id of unknown) {
    primeInFlight.delete(nexusKey(game, id))
    if (got && (!got.error || found[`${id}`])) {
      primeAsked.add(nexusKey(game, id))
    }
  }
  addDetails(game, unknown, found)
}

let watching = false

// The mods a failed prime left out are asked for again the moment Nexus answers again.
function watchNexusReachable() {
  if (!watching) {
    watching = true
    Events.On('netstate:changed', () => {
      nexusChanged().catch(reportUnexpected)
    })
  }
}

async function nexusChanged() {
  const states = await readStates()
  if (states?.some((st) => st.id === 'nexus' && !st.unreachable)) {
    await primeDetails(primed.game, primed.ids)
  }
}

// Whether the entry holds a mod's whole details, not just the batched page data.
const isFull = (entry: Entry | undefined) => Boolean(entry?.details && !entry.details.partial)

// The entry for one mod, read on first use; undefined while the first read is on its way. Page data from the batch is
// not enough for a caller that asks for one mod, so it reads as missing until the whole details arrive.
const useNexusEntry = (game: string, id: number) => {
  useEffect(() => {
    if (game && id) {
      enqueue(game, id)
    }
  }, [game, id])
  const entry = useNexusDetails((s) => nexusEntryOf(s.byKey, game, id))
  return isFull(entry) ? entry : { ...entry, details: undefined }
}

// A mod's page data (status, version, endorsements) for a list row: whatever the batch or a full read gave, and no
// request of its own.
const useNexusPage = (game: string, id: number) =>
  useNexusDetails((s) => nexusEntryOf(s.byKey, game, id)?.details?.page)

// Whether the mod's Nexus page offers a download, as a flag: page data arrives for every card at once.
const useNexusDownloadable = (game: string, id: number) =>
  useNexusDetails((s) => {
    const page = nexusEntryOf(s.byKey, game, id)?.details?.page
    return offersNexusDownload(page?.status, page?.available)
  })

interface SeenWatermark {
  newestFileUnix: number
  newestChange: string
}

const MS_PER_SEC = 1000
const LEGACY_NEXUS_KEY = /^\d+$/
const NEXUS_SEEN_KEY = /^nexus:(.+):(\d+)$/

const fileUnix = (iso: string) => {
  const t = Date.parse(iso)
  return Number.isNaN(t) ? 0 : Math.floor(t / MS_PER_SEC)
}

const watermarkOf = (details: Pick<Details, 'files' | 'changelogs'>): SeenWatermark => {
  let newestFileUnix = 0
  for (const f of currentFiles(details.files ?? [], 0)) {
    newestFileUnix = Math.max(newestFileUnix, fileUnix(f.uploaded))
  }
  if (newestFileUnix === 0) {
    for (const f of details.files ?? []) {
      newestFileUnix = Math.max(newestFileUnix, fileUnix(f.uploaded))
    }
  }
  let newestChange = ''
  for (const c of details.changelogs ?? []) {
    if (!newestChange || isNewer(c.version, newestChange)) {
      newestChange = c.version
    }
  }
  return { newestFileUnix, newestChange }
}

const isNewSinceLooked = (seen: SeenWatermark | undefined, now: SeenWatermark) => {
  if (!seen) {
    return false
  }
  if (now.newestFileUnix > seen.newestFileUnix) {
    return true
  }
  if (!now.newestChange || now.newestChange === seen.newestChange) {
    return false
  }
  if (!seen.newestChange) {
    return true
  }
  return isNewer(now.newestChange, seen.newestChange)
}

const fileIsNewSinceLooked = (uploaded: string, seen: SeenWatermark | undefined) =>
  Boolean(seen && fileUnix(uploaded) > seen.newestFileUnix)

const changelogIsNewSinceLooked = (version: string, seen: SeenWatermark | undefined) => {
  if (!(seen && version)) {
    return false
  }
  if (!seen.newestChange) {
    return true
  }
  return isNewer(version, seen.newestChange)
}

// byKey is per game and mod id; legacy holds what releases before the per-game key recorded by mod id alone, which
// counts for any game that has no look of its own at that id.
const useNexusSeen = create<{
  byKey: Record<string, SeenWatermark | undefined>
  legacy: Record<number, SeenWatermark | undefined>
}>(() => ({ byKey: {}, legacy: {} }))

const seenOf = (s: ReturnType<typeof useNexusSeen.getState>, game: string, id: number) =>
  s.byKey[nexusKey(game, id)] ?? s.legacy[id]

// The newest version the user has looked at for each mod of a source other than Nexus, keyed by the favourites key.
const useSourceSeen = create<{ bySource: Record<string, string | undefined> }>(() => ({
  bySource: {},
}))

const newestOffered = (installed: string, offered: string | undefined) =>
  offered && isNewer(offered, installed) ? offered : installed

// Whether the source offers a version newer than the one last looked at; a mod never looked at is not new.
const isNewSource = (seen: string | undefined, offered: string | undefined) =>
  Boolean(seen && offered && isNewer(offered, seen))

const markSourceLooked = (key: string, version: string) => {
  if (!(key && version) || useSourceSeen.getState().bySource[key] === version) {
    return
  }
  useSourceSeen.setState((s) => ({ bySource: { ...s.bySource, [key]: version } }))
  MarkSeen(key, 0, version).catch(reportUnexpected)
}

const initNexusSeen = () =>
  Seen()
    .then((snap) => {
      const byKey: Record<string, SeenWatermark | undefined> = {}
      const legacy: Record<number, SeenWatermark | undefined> = {}
      const bySource: Record<string, string> = {}
      for (const [key, entry] of Object.entries(snap ?? {})) {
        if (!entry) {
          continue
        }
        const w = { newestFileUnix: entry.newestFileUnix, newestChange: entry.newestChange }
        const nexus = NEXUS_SEEN_KEY.exec(key)
        if (nexus) {
          byKey[nexusKey(nexus[1] ?? '', Number(nexus[2]))] = w
        } else if (LEGACY_NEXUS_KEY.test(key)) {
          legacy[Number(key)] = w
        } else {
          bySource[key] = entry.newestChange
        }
      }
      useNexusSeen.setState({ byKey, legacy })
      useSourceSeen.setState({ bySource })
    })
    .catch(reportUnexpected)

const markLooked = (game: string, modId: number, details: Details) => {
  const w = watermarkOf(details)
  useNexusSeen.setState((s) => ({ byKey: { ...s.byKey, [nexusKey(game, modId)]: w } }))
  MarkNexusSeen(game, modId, w.newestFileUnix, w.newestChange).catch(reportUnexpected)
}

const useLookedSnapshot = (game: string, modId: number, details: Details | undefined) => {
  const [looked] = useState(() => seenOf(useNexusSeen.getState(), game, modId))
  useEffect(() => {
    if (details && !details.partial && game && modId) {
      markLooked(game, modId, details)
    }
  }, [game, modId, details])
  return looked
}

// The answer, not the details: details arrive for every card at once, and a card should redraw only when it changes.
const useNexusFresh = (game: string, nexusId: number) => {
  const seen = useNexusSeen((s) => seenOf(s, game, nexusId))
  return useNexusDetails((s) => {
    const details = nexusEntryOf(s.byKey, game, nexusId)?.details
    return nexusId !== 0 && details !== undefined && isNewSinceLooked(seen, watermarkOf(details))
  })
}

export type { NexusById }
export {
  changelogIsNewSinceLooked,
  detailsFor,
  fileIsNewSinceLooked,
  initNexusSeen,
  isFull,
  isNewSinceLooked,
  isNewSource,
  loadDetails,
  markSourceLooked,
  newestOffered,
  nexusChanged,
  nexusEntryOf,
  primeDetails,
  useGameNexusDetails,
  useLookedSnapshot,
  useNexusDetails,
  useNexusDownloadable,
  useNexusEntry,
  useNexusFresh,
  useNexusPage,
  useNexusSeen,
  useSourceSeen,
  watermarkOf,
}

// Fills the details store from the on-disk Nexus cache only, so Update review never hits the network.
export async function mergeCachedDetails(game: string, ids: number[]): Promise<void> {
  const want = [...new Set(ids)].filter((id) => id > 0)
  const { byKey } = useNexusDetails.getState()
  const unknown = want.filter((id) => !nexusEntryOf(byKey, game, id)?.details)
  if (!game || unknown.length === 0) {
    return
  }
  addDetails(game, unknown, (await CachedDetails(game, unknown)) ?? {})
}
