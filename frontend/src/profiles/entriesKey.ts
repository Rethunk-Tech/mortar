import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

/** The profile's entries and their mods' versions as a string: equal across a save that only switches mods on or off. */
export const entriesKey = (p: Pick<Profile, 'id' | 'entries'>) =>
  `${p.id}|${(p.entries ?? []).map((e) => `${e.key}:${(e.mods ?? []).map((m) => `${m.id}@${m.version}`).join(',')}`).join('|')}`
