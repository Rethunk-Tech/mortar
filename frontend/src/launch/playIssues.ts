import type { Group } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/playcheck/models.ts'
import { Check } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/playcheck/service.ts'
import { listNames } from '../i18n/list.ts'
import { itemLabel } from '../profiles/historyDiff.ts'
import { saveName } from '../saves/saveName.ts'

// Go decides which warnings there are and what they list; the window words the parts that depend on the language:
// a conflict's mods, a change, a save.
type PlayIssueGroup = Omit<Group, 'names'> & { names: string[]; save: string }

function worded(g: Group): PlayIssueGroup {
  const names = g.names ?? []
  const changes = g.changes ?? []
  return {
    ...g,
    names:
      changes.length > 0
        ? changes.map(itemLabel)
        : names.map((name, i) => {
            const parts = g.nameParts?.[i] ?? []
            return parts.length > 0 ? listNames(parts) : name
          }),
    save:
      g.kind === 'lastProfile'
        ? saveName({ farm: g.saveFarm ?? '', folder: g.saveFolder ?? '', name: g.saveName })
        : '',
  }
}

async function gatherPlayIssues(game: string, profileId: string): Promise<PlayIssueGroup[]> {
  return ((await Check(game, profileId)) ?? []).map(worded)
}

function overflowIssueCount(group: Pick<PlayIssueGroup, 'count' | 'names'>): number {
  return Math.max(0, group.count - group.names.length)
}

export type { PlayIssueGroup }
export { gatherPlayIssues, overflowIssueCount, worded }
