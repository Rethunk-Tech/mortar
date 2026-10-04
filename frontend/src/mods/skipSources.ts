import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'

interface SourceGroups {
  ignore: Map<string, Mod[]>
  use: Map<string, Mod[]>
}

interface SkipEntry {
  key: string
  skipSources?: string[] | null
}

function push(groups: Map<string, Mod[]>, source: string, mod: Mod) {
  groups.set(source, [...(groups.get(source) ?? []), mod])
}

// Ignore: each selected mod's update source that is not ignored yet. Use: every source a selected mod ignores.
function skipSourceGroups(
  selected: Mod[],
  updates: readonly { key: string; source?: string }[],
  entries: readonly SkipEntry[] | null | undefined,
): SourceGroups {
  const groups: SourceGroups = { ignore: new Map(), use: new Map() }
  for (const mod of selected) {
    const skipped = entries?.find((entry) => entry.key === mod.key)?.skipSources ?? []
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
