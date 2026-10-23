import type { SMAPIProblem } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'

// The token internal/problems stores for a dismissed run problem (runProblemToken): its kind and mod, not its text,
// which carries versions a rebuild changes.
export const runProblemMod = (p: SMAPIProblem) => p.modId || p.modName
export const runProblemToken = (p: SMAPIProblem) => `runProblem\t${p.kind}\t${runProblemMod(p)}`
