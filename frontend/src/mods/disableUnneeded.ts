import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { openOverride } from '../profiles/openOverrides.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'
import { dependentsOf, idKey, requiredIdsOf } from './dependents.ts'
import { readsImplicit } from './installReason.ts'
import { entryOf } from './lookup.ts'

const requires = (a: Mod, b: Mod) => requiredIdsOf(a).some((id) => idKey(id) === idKey(b.id))
// What a mod requires or can use: its needs include its optional dependencies.
const uses = (a: Mod, b: Mod) =>
  [...(a.needs ?? []), a.contentPackFor ?? ''].some((id) => idKey(id) === idKey(b.id))

/**
 * The implicit mods still on that only the mods just switched off needed: what those mods require, followed through
 * what that requires in turn, each implicit and needed by no mod that stays on. When the mods are being removed, a
 * switched-off library goes too, and a switched-off mod that stays still needs what it requires. A mod that stays on
 * keeps what it can use as well, since removing an optional dependency takes away the feature it adds.
 */
export function unneededImplicit(
  all: readonly Mod[],
  off: readonly Mod[],
  implicit: (m: Mod) => boolean,
  removing = false,
): Mod[] {
  const byId = new Map(all.map((m) => [idKey(m.id), m]))
  const offIds = new Set(off.map((m) => idKey(m.id)))
  const candidates: Mod[] = []
  const queue = off.flatMap((m) => requiredIdsOf(m))
  for (let id = queue.shift(); id !== undefined; id = queue.shift()) {
    const dep = byId.get(idKey(id))
    if (
      dep &&
      (removing || dep.enabled) &&
      implicit(dep) &&
      !offIds.has(idKey(dep.id)) &&
      !candidates.includes(dep)
    ) {
      candidates.push(dep)
      queue.push(...requiredIdsOf(dep))
    }
  }
  const out: Mod[] = []
  for (let changed = true; changed; ) {
    changed = false
    for (const c of candidates) {
      const stillNeeded = all.some(
        (m) => !offIds.has(idKey(m.id)) && (m.enabled ? uses(m, c) : removing && requires(m, c)),
      )
      if (!(offIds.has(idKey(c.id)) || stillNeeded)) {
        offIds.add(idKey(c.id))
        out.push(c)
        changed = true
      }
    }
  }
  return out
}

// What to do, as the open profile sees it, with what a disable leaves behind: its own override, else the game's setting.
function disableDependenciesMode(): string {
  return openOverride(
    'disableDependencies',
    gamePrefs(useSettings.getState()).disableDependencies || 'ask',
  )
}

/**
 * What switching `off` off leaves behind: the mods still on that need them, and the implicit mods still on that only
 * they needed, counting the dependents as switched off too when `withDependents`.
 */
export function disableFallout(all: readonly Mod[], off: readonly Mod[], withDependents = true) {
  const profile = openProfileOf(useProfiles.getState())
  const dependents = dependentsStillOn(all, off)
  const unneeded = unneededImplicit(all, withDependents ? [...off, ...dependents] : off, (m) =>
    readsImplicit(entryOf(profile, m.key)?.implicit, m, all),
  )
  return { dependents, unneeded }
}

/**
 * What else switching `off` off takes, per the Disable dependencies preference: null to ask first, else the mods to
 * switch off with them (none when nothing needs them and they leave no library behind, or the preference is Never).
 */
export function disableAlso(all: readonly Mod[], off: readonly Mod[]): Mod[] | null {
  const mode = disableDependenciesMode()
  if (mode === 'never') {
    return []
  }
  const { dependents, unneeded } = disableFallout(all, off)
  if (dependents.length + unneeded.length === 0) {
    return []
  }
  return mode === 'ask' ? null : [...dependents, ...unneeded]
}

/**
 * The mods still on that need the mods just switched off, and the mods that need those in turn: with a required mod
 * off they cannot work, so one click can switch the whole chain off.
 */
export function dependentsStillOn(all: readonly Mod[], off: readonly Mod[]): Mod[] {
  const out: Mod[] = []
  for (
    let next = dependentsOf(all, off);
    next.length > 0;
    next = dependentsOf(all, [...off, ...out])
  ) {
    out.push(...next)
  }
  return out
}
