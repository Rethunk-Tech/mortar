import { useDetail } from './detail.ts'
import { useSelection } from './selection.ts'

/** Whether mod id is marked: in the selection, or the open mod while nothing is selected. Each row selects only its
 * own answer, so opening or selecting a mod re-renders the rows whose mark changed, not every row on screen. */
export function useMarked(id: string): boolean {
  const selected = useSelection((s) => s.ids.includes(id))
  const none = useSelection((s) => s.ids.length === 0)
  const open = useDetail((s) => s.detailId === id)
  return selected || (none && open)
}

/** Whether mod id is its list's one Tab stop: the open mod's row, or the first row when the open mod is not listed. */
export function useTabStop(orderedIds: readonly string[], id: string): boolean {
  return useDetail((s) =>
    orderedIds.includes(s.detailId) ? s.detailId === id : orderedIds[0] === id,
  )
}
