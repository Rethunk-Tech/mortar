import type { SourceInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/migrate/models.ts'

// The chooser is for a game Vortex supports when none of the detected sources is a Vortex install.
export function offerVortexFolder(supported: boolean, sources: SourceInfo[]): boolean {
  return supported && !sources.some((s) => s.kind === 'vortex')
}

// The same for Mod Organizer 2, whose instance may live anywhere.
export function offerMO2Folder(supported: boolean, sources: SourceInfo[]): boolean {
  return supported && !sources.some((s) => s.kind === 'mo2')
}
