import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'

// The launch screen's log box: lines for each mod loading are counted by the loader's mark instead, and debug and trace
// lines stay hidden as in the Console by default, so the box does not flash with a mod's bookkeeping.
export function launchLogLines(entries: readonly Entry[], limit: number): Entry[] {
  return entries
    .filter((e) => !(e.load || e.level === Level.Debug || e.level === Level.Trace))
    .slice(-limit)
}
