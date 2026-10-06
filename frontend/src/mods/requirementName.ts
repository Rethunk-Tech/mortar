import { localId } from './dependents.ts'
import { problemsOf, sameId } from './lookup.ts'
import { useMods } from './store.ts'

// A mod named by its id: an installed copy's name, else the page the Problems check found for it, else the bare id.
export function useRequirementName(id: string): string {
  return useMods((s) => {
    const installed = s.mods.find((m) => sameId(localId(m.id), localId(id)))?.name
    if (installed) {
      return installed
    }
    const hit = problemsOf(s.problems).find(
      (p) => p.kind === 'missing' && sameId(localId(p.missing.id), localId(id)),
    )
    const page = hit?.kind === 'missing' ? (hit.missing.where?.pageName?.trim() ?? '') : ''
    return page || localId(id)
  })
}
