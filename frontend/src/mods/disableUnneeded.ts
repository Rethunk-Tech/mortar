import { msg, plural } from '@lingui/core/macro'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
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

// After mods are switched off, offers to switch off the implicit mods only they needed.
export function offerUnneeded(
  all: readonly Mod[],
  off: readonly Mod[],
  switchOff: (mods: Mod[]) => Promise<void>,
) {
  const profile = openProfileOf(useProfiles.getState())
  const unneeded = unneededImplicit(all, off, (m) =>
    readsImplicit(entryOf(profile, m.key)?.implicit, m, all),
  )
  if (unneeded.length === 0) {
    return
  }
  useToasts.getState().push({
    kind: 'info',
    title: plural(
      { count: unneeded.length },
      {
        one: '# library only the disabled mods needed is still enabled',
        other: '# libraries only the disabled mods needed are still enabled',
      },
    ),
    action: {
      label: i18n._(msg`Disable`),
      run: () => switchOff(unneeded).catch(reportUnexpected),
    },
  })
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

// After mods are switched off, warns about the mods that need them and offers to switch those off too.
export function offerDependents(
  all: readonly Mod[],
  off: readonly Mod[],
  switchOff: (mods: Mod[]) => Promise<void>,
) {
  const dependents = dependentsStillOn(all, off)
  if (dependents.length === 0) {
    return
  }
  useToasts.getState().push({
    kind: 'warning',
    title: plural(
      { count: dependents.length },
      {
        one: '# enabled mod needs a mod you disabled',
        other: '# enabled mods need a mod you disabled',
      },
    ),
    body: listNames(dependents.map((m) => m.name)),
    action: {
      label: i18n._(msg`Disable them too`),
      run: () => switchOff(dependents).catch(reportUnexpected),
    },
  })
}
