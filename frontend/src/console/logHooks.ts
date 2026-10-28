import { useLingui } from '@lingui/react/macro'
import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import type { HiddenLogSource } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { collapseRepeats, type Filters, shownLog, visible } from './filter.ts'
import { useHiddenSources } from './hiddenSources.ts'
import { withoutHidden } from './logSource.ts'
import { useConsole } from './store.ts'

function useMine() {
  return useConsole((s) => s.shown.profile) === useProfiles((s) => s.openId)
}

// Each component asks for the same lists on every line that arrives; the last answer is shared among them.
function lastResult<A extends readonly unknown[], R>(compute: (...args: A) => R) {
  let args: A | null = null
  let result: R
  return (...next: A): R => {
    if (!(args && args.length === next.length && args.every((a, i) => a === next[i]))) {
      result = compute(...next)
      args = next
    }
    return result
  }
}

const shownEntries = lastResult(
  (entries: Entry[], mine: boolean, hidden: readonly HiddenLogSource[]) =>
    withoutHidden(shownLog(entries, mine), hidden),
)
const visibleRows = lastResult((entries: Entry[], filters: Filters) =>
  collapseRepeats(visible(entries, filters)),
)

// The log as shown everywhere (lines, level counts, copies): without what the player hid by its source.
export function useShownEntries() {
  const entries = useConsole((s) => s.entries)
  const mine = useMine()
  const game = useConsole((s) => s.shown.game)
  const { hidden } = useHiddenSources(game)
  return shownEntries(entries, mine, hidden)
}

export function useVisible() {
  const entries = useShownEntries()
  const filters = useConsole((s) => s.filters)
  return visibleRows(entries, filters)
}

export function useLevelNames(): Record<Level, string> {
  const { t } = useLingui()
  return {
    [Level.$zero]: '',
    [Level.Trace]: t`Trace`,
    [Level.Debug]: t`Debug`,
    [Level.Info]: t`Info`,
    [Level.Warn]: t`Warn`,
    [Level.Error]: t`Error`,
    [Level.Alert]: t`Alert`,
  }
}
