import { create } from 'zustand'

function clickRow(id: string): SelectionState {
  return { ids: [id], anchor: id }
}

export interface SelectionState {
  ids: readonly string[]
  anchor: string | null
}

export function getInitialState(): SelectionState {
  return { ids: [], anchor: null }
}

export function toggleRow(state: SelectionState, id: string): SelectionState {
  const has = state.ids.includes(id)
  const ids = has ? state.ids.filter((x) => x !== id) : [...state.ids, id]
  return { ids, anchor: id }
}

export function rangeRows(
  state: SelectionState,
  orderedIds: readonly string[],
  id: string,
): SelectionState {
  const anchor = state.anchor ?? id
  const a = orderedIds.indexOf(anchor)
  const b = orderedIds.indexOf(id)
  if (a < 0 || b < 0) {
    return clickRow(id)
  }
  const lo = Math.min(a, b)
  const hi = Math.max(a, b)
  return { ids: orderedIds.slice(lo, hi + 1), anchor }
}

export function selectAllRows(orderedIds: readonly string[]): SelectionState {
  return { ids: orderedIds, anchor: orderedIds[0] ?? null }
}

export function clearSelection(): SelectionState {
  return getInitialState()
}

export function applyClick(
  state: SelectionState,
  orderedIds: readonly string[],
  id: string,
  mods: { shiftKey: boolean; ctrlKey: boolean; metaKey: boolean },
): SelectionState {
  if (mods.shiftKey) {
    return rangeRows(state, orderedIds, id)
  }
  if (mods.ctrlKey || mods.metaKey) {
    return toggleRow(state, id)
  }
  return clickRow(id)
}

export const useSelection = create<
  SelectionState & {
    click: (
      orderedIds: readonly string[],
      id: string,
      mods: { shiftKey: boolean; ctrlKey: boolean; metaKey: boolean },
    ) => void
    selectAll: (orderedIds: readonly string[]) => void
    clear: () => void
    prune: (visibleIds: readonly string[]) => void
  }
>((set, get) => ({
  ...getInitialState(),
  click: (orderedIds, id, mods) => set(applyClick(get(), orderedIds, id, mods)),
  selectAll: (orderedIds) => set(selectAllRows(orderedIds)),
  clear: () => set(clearSelection()),
  prune: (visibleIds) => {
    const allowed = new Set(visibleIds)
    const { ids: current, anchor } = get()
    const ids = current.filter((id) => allowed.has(id))
    const nextAnchor = anchor !== null && allowed.has(anchor) ? anchor : (ids[0] ?? null)
    if (
      ids.length === current.length &&
      ids.every((id, i) => id === current[i]) &&
      nextAnchor === anchor
    ) {
      return
    }
    set({ ids, anchor: nextAnchor })
  },
}))
