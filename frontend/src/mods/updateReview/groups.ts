import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type {
  Mod,
  Profile,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { cmpText } from '../cmpText.ts'
import { readsImplicit } from '../installReason.ts'
import { entryOf, sameId } from '../lookup.ts'

const FIRST = ['github', 'thunderstore']

/** Nexus's own name in an update's source: a SMAPI channel update says "Nexus", a source driver "Nexus Mods". */
export const isNexusSource = (name: string) => name.toLowerCase().startsWith('nexus')

/** Whether a Nexus-installed mod's update was reported by a site other than Nexus. */
export const reportsElsewhere = (installedFrom: string | undefined, source: string) =>
  installedFrom === 'nexus' && source !== '' && !isNexusSource(source)

const rank = (name: string) => {
  if (isNexusSource(name)) {
    return 0
  }
  const i = FIRST.findIndex((f) => name.toLowerCase().startsWith(f))
  return i === -1 ? FIRST.length + 1 : i + 1
}

/** Rows grouped by the source that reported the update: Nexus, GitHub, Thunderstore, then the rest alphabetically. */
export function groupBySource(list: readonly Update[]): { name: string; list: Update[] }[] {
  const by = Map.groupBy(list, (u) => u.source)
  return [...by.entries()]
    .map(([name, rows]) => ({ name, list: rows }))
    .sort((a, b) => rank(a.name) - rank(b.name) || cmpText(a.name, b.name))
}

/** The review's lists: the updates of implicit mods as Libraries, and the rest by the source that reported them. */
export function splitBySource(
  list: readonly Update[],
  mods: readonly Mod[],
  profile: Profile,
): { groups: { name: string; list: Update[] }[]; libraries: Update[] } {
  const libraries: Update[] = []
  const rest: Update[] = []
  for (const u of list) {
    const mod = mods.find((m) => m.key === u.key && sameId(m.id, u.id)) ?? { key: u.key, id: u.id }
    if (readsImplicit(entryOf(profile, u.key)?.implicit, mod, mods)) {
      libraries.push(u)
    } else {
      rest.push(u)
    }
  }
  return { groups: groupBySource(rest), libraries }
}
