import type { Run } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { Runs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { shareInFlight } from '../profiles/listProfiles.ts'

const shared = shareInFlight<Run[]>()

/** A profile's recorded runs. The console's picker, empty state and toolbar ask together when it opens, so callers
 * asking while a read is under way share it. */
export function listRuns(game: string, profile: string): Promise<Run[]> {
  return shared(`${game}\n${profile}`, async () => (await Runs(game, profile)) ?? [])
}
