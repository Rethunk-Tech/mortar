import type { SMAPIProblem } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { sameId } from '../mods/lookup.ts'

type LiveMod = Pick<Mod, 'uniqueId' | 'name' | 'version' | 'enabled'>

// SMAPI names a mod as "<name> <version>" in its log, so the match tries the id, the bare name, then the
// longest profile mod name that the logged name starts with.
function matchMod(problem: SMAPIProblem, mods: readonly LiveMod[]): LiveMod | undefined {
  if (problem.modId !== '') {
    const byId = mods.find((m) => sameId(m.uniqueId, problem.modId))
    if (byId) {
      return byId
    }
  }
  const logged = problem.modName
  if (logged === '') {
    return undefined
  }
  const exact = mods.find((m) => m.name === logged)
  if (exact) {
    return exact
  }
  let best: LiveMod | undefined
  for (const m of mods) {
    if (
      m.name !== '' &&
      logged.startsWith(`${m.name} `) &&
      m.name.length > (best?.name.length ?? 0)
    ) {
      best = m
    }
  }
  return best
}

function loggedVersion(problem: SMAPIProblem, mod: LiveMod): string {
  const rest = problem.modName.startsWith(`${mod.name} `)
    ? problem.modName.slice(mod.name.length + 1).trim()
    : ''
  return rest.split(' ')[0] ?? ''
}

// A run's problem still applies only while the profile looks as it did in that run: the mod is still there,
// at the logged version, and the offered fix has not already been made.
export function stillApplies(problem: SMAPIProblem, mods: readonly LiveMod[]): boolean {
  if (problem.fix === 'installDependency') {
    const dep = problem.dependency ?? ''
    return dep === '' || !mods.some((m) => sameId(m.uniqueId, dep))
  }
  const mod = matchMod(problem, mods)
  if (!mod) {
    return false
  }
  const version = loggedVersion(problem, mod)
  if (version !== '' && mod.version !== '' && version !== mod.version) {
    return false
  }
  if (problem.fix === 'disable' && !mod.enabled) {
    return false
  }
  return true
}
