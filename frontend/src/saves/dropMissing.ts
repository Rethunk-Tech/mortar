import type { Fit } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'

export function dropMissing(fits: Fit[], uniqueId: string, folder?: string): Fit[] {
  return fits.map((f) =>
    folder !== undefined && f.folder !== folder
      ? f
      : {
          ...f,
          missing: (f.missing ?? []).filter((m) => m.uniqueId !== uniqueId),
        },
  )
}
