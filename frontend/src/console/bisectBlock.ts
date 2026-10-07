import { Outcome } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import type { Run } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'

// A run the player stopped counts only when it crashed before the stop.
function runCrashed(run: Run): boolean {
  return run.outcome === Outcome.OutcomeCrashed || (run.errors > 0 && !run.exit?.stopped)
}

export type BisectBlock = 'no-profile' | 'no-runs' | 'ended-normally' | 'cause-known'

// Why a crash check cannot start for the profile's runs (newest first), or null when it can.
export function bisectBlock(runs: readonly Run[] | null | undefined): BisectBlock | null {
  const latest = runs?.[0]
  if (!latest) {
    return 'no-runs'
  }
  if (!runCrashed(latest)) {
    return 'ended-normally'
  }
  return latest.cause === null || latest.cause === undefined ? null : 'cause-known'
}

export type BisectFailure = 'no-mods' | 'missing-files' | 'other'

// The crash check's Go errors, as the plain cases the dialog can explain.
export function bisectFailure(error: string | undefined): BisectFailure {
  if (error?.includes('no enabled user mods')) {
    return 'no-mods'
  }
  if (error?.includes('is missing')) {
    return 'missing-files'
  }
  return 'other'
}
