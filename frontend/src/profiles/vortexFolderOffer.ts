import type { SourceInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/migrate/models.ts'

// The chooser is for a game Vortex supports when none of the detected sources is a Vortex install.
export function offerVortexFolder(supported: boolean, sources: SourceInfo[]): boolean {
  return supported && !sources.some((s) => s.kind === 'vortex')
}
