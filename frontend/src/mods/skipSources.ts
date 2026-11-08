import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { entryOf } from './lookup.ts'

interface SourceGroups {
  ignore: Map<string, Mod[]>
  use: Map<string, Mod[]>
}

function push(groups: Map<string, Mod[]>, source: string, mod: Mod) {
  const list = groups.get(source)
  if (list) {
    list.push(mod)
  } else {
    groups.set(source, [mod])
  }
}

// Ignore: each selected mod's update source that is not ignored yet. Use: every source a selected mod ignores.
function skipSourceGroups(
  selected: Mod[],
  updates: readonly { key: string; source?: string }[],
  profile: Profile | null | undefined,
): SourceGroups {
  const groups: SourceGroups = { ignore: new Map(), use: new Map() }
  for (const mod of selected) {
    const skipped = entryOf(profile, mod.key)?.skipSources ?? []
    const source = updates.find((update) => update.key === mod.key)?.source
    if (source && !skipped.includes(source)) {
      push(groups.ignore, source, mod)
    }
    for (const ignored of skipped) {
      push(groups.use, ignored, mod)
    }
  }
  return groups
}

export { type SourceGroups, skipSourceGroups }
