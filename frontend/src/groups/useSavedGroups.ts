import { useLingui } from '@lingui/react/macro'
import { useCallback } from 'react'
import type { SavedGroup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/models.ts'
import { List } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/service.ts'
import { getLast } from '../shell/lastGood.ts'
import { useLastGood } from '../shell/useLastGood.ts'
import { useOnWindowFocus } from '../shell/useOnWindowFocus.ts'
import { reportError } from '../toasts/report.ts'

const NONE: SavedGroup[] = []

// The game's saved groups; `open` false skips the read (a closed dialog). The last list stays on screen while a
// refresh runs, so `loading` is true only before the first list ever arrived.
export function useSavedGroups(game: string, open = true) {
  const { t } = useLingui()
  const key = `saved-groups:${game}`
  const { data, error, set, reload } = useLastGood<SavedGroup[]>(
    key,
    open ? () => List(game).then((listed) => listed ?? NONE) : null,
    [game, open],
    { onError: reportError(t`Could not read saved groups`) },
  )
  // Groups saved from the command line show once the window is back in front.
  useOnWindowFocus(reload, open)
  const setSavedGroups = useCallback(
    (update: (current: SavedGroup[]) => SavedGroup[]) =>
      set(update(getLast<SavedGroup[]>(key) ?? NONE)),
    [key, set],
  )
  return {
    savedGroups: data ?? NONE,
    setSavedGroups,
    loading: open && data === null && error === null,
  }
}
