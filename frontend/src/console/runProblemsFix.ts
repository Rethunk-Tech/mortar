import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'

/** Which copies a Remove duplicate action keeps versus deletes from the profile. */
export function duplicateCopies(mods: Mod[]): { keep?: Mod; remove: Mod[] } {
  if (mods.length > 1) {
    const [keep, ...remove] = mods
    return keep ? { keep, remove } : { remove }
  }
  return { remove: mods }
}
