import { readStored, writeStored } from '../shell/useStoredState.ts'

const KEY = 'mortar.lanResume'

// Where a share came from: the sending computer and its profile, so a later share of the same profile can update
// what this computer made of the first.
interface LanOrigin {
  game: string
  senderId: string
  profileId: string
}

interface Listed {
  id: string
  name: string
}

type IncomingChoice = { kind: 'update'; profileId: string; name: string } | { kind: 'new' }

const isMap = (v: unknown): v is Record<string, string> =>
  typeof v === 'object' &&
  v !== null &&
  !Array.isArray(v) &&
  Object.values(v).every((x) => typeof x === 'string')

const keyOf = (o: LanOrigin) => JSON.stringify([o.game, o.senderId, o.profileId])

function lanOriginOf(arrival: {
  game: string
  senderId?: string
  profileId?: string
}): LanOrigin | undefined {
  const { game, senderId, profileId } = arrival
  return senderId && profileId ? { game, senderId, profileId } : undefined
}

function rememberLanProfile(origin: LanOrigin, localId: string): void {
  writeStored(KEY, { ...readStored(KEY, {}, isMap), [keyOf(origin)]: localId })
}

// The local profile an earlier share of the same sender profile went into, unless it has been deleted since.
function mappedProfile(
  origin: LanOrigin | undefined,
  profiles: readonly Listed[],
): Listed | undefined {
  if (!origin) {
    return undefined
  }
  const id = readStored(KEY, {}, isMap)[keyOf(origin)]
  return profiles.find((p) => p.id === id)
}

// Resuming a profile updates its copy; anything else is a new profile.
function choose(mapped: Listed | undefined): IncomingChoice {
  return mapped ? { kind: 'update', profileId: mapped.id, name: mapped.name } : { kind: 'new' }
}

export type { LanOrigin }
export { choose, lanOriginOf, mappedProfile, rememberLanProfile }
