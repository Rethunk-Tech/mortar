import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { authorFieldIncludes } from './authorNormalize.ts'
import { cmpText } from './cmpText.ts'
import { idKey } from './dependents.ts'
import { sameId } from './lookup.ts'

interface AuthorModProfile {
  profileId: string
  profileName: string
  version: string
  enabled: boolean
}

interface AuthorModRow {
  uniqueId: string
  name: string
  profiles: AuthorModProfile[]
}

function addHit(
  byId: Map<string, AuthorModRow>,
  row: AuthorModProfile,
  mod: { uniqueId: string; name: string },
) {
  const key = idKey(mod.uniqueId)
  const existing = byId.get(key)
  if (existing) {
    if (existing.name === '') {
      existing.name = mod.name
    }
    existing.profiles.push(row)
    return
  }
  byId.set(key, { uniqueId: mod.uniqueId, name: mod.name, profiles: [row] })
}

function collectFromProfile(byId: Map<string, AuthorModRow>, profile: Profile, author: string) {
  for (const entry of profile.entries ?? []) {
    for (const mod of entry.mods ?? []) {
      if (authorFieldIncludes(mod.author, author)) {
        const enabled = !(entry.disabled ?? []).some((id) => sameId(id, mod.uniqueId))
        addHit(
          byId,
          {
            profileId: profile.id,
            profileName: profile.name,
            version: mod.version,
            enabled,
          },
          mod,
        )
      }
    }
  }
}

function modsByAuthor(profiles: Profile[], author: string): AuthorModRow[] {
  const byId = new Map<string, AuthorModRow>()
  for (const profile of profiles) {
    if (!profile.error) {
      collectFromProfile(byId, profile, author)
    }
  }
  const out = [...byId.values()]
  for (const mod of out) {
    mod.profiles.sort((a, b) => cmpText(a.profileName, b.profileName))
  }
  out.sort((a, b) => {
    const byName = cmpText(a.name, b.name)
    if (byName !== 0) {
      return byName
    }
    return cmpText(a.uniqueId, b.uniqueId)
  })
  return out
}

export { modsByAuthor }
