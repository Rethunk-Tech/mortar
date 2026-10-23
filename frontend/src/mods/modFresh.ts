import { useEffect } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { entryFavouriteKey } from './favourites.ts'
import { entryOf, nexusIdOf, updateFor } from './lookup.ts'
import {
  isNewSource,
  markSourceLooked,
  newestOffered,
  useNexusFresh,
  useSourceSeen,
} from './nexusDetails.ts'
import { useUpdates } from './updates.ts'

// The key of a mod from a source other than Nexus, whose "new since you looked" mark follows the version it offers.
const sourceKeyOf = (profile: Profile, mod: Mod) => {
  const entry = entryOf(profile, mod.key)
  return entry?.source.kind === 'nexus' ? '' : entryFavouriteKey(entry)
}

// Whether the mod's source has something new since the user last looked at it.
export function useModFresh(profile: Profile, mod: Mod) {
  const nexus = useNexusFresh(nexusIdOf(profile, mod))
  const key = sourceKeyOf(profile, mod)
  const offered = useUpdates((s) => updateFor(s.updates, mod, profile))?.version
  // The answer, not the seen version: opening the panel records a look, which must not redraw it when nothing is new.
  const fresh = useSourceSeen((s) => key !== '' && isNewSource(s.bySource[key], offered))
  return nexus || fresh
}

// Opening a mod's panel counts as looking at the newest version its source offers.
export function useSourceLooked(profile: Profile, mod: Mod) {
  const key = sourceKeyOf(profile, mod)
  const offered = useUpdates((s) => updateFor(s.updates, mod, profile))?.version
  useEffect(() => {
    markSourceLooked(key, newestOffered(mod.version, offered))
  }, [key, mod.version, offered])
}
