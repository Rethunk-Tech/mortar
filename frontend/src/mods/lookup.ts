import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'

export const siblingsOf = (mods: Mod[], mod: Mod) =>
  mods.filter((m) => m.key === mod.key && m.uniqueId !== mod.uniqueId)

export const sourceKind = (profile: Profile, mod: Mod) =>
  (profile.entries ?? []).find((e) => e.key === mod.key)?.source.kind ?? ''
