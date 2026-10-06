import { create } from 'zustand'
import type { MenuAnchor } from '../mods/menu.ts'
import type { BrowseItem } from './browseTypes.ts'

const hitKey = (item: Pick<BrowseItem, 'source' | 'id'>) => `${item.source}:${item.id}`

interface Picked {
  item: BrowseItem
  // The source the card had picked when it was opened; the panel's own badges pick from there.
  source: string
}

/** The Browse hit whose details panel is open, and the hit a right-click menu is open for. */
const useBrowseSelection = create<{
  selected: Picked | null
  menu: (Picked & { anchor: MenuAnchor }) | null
  select: (picked: Picked | null) => void
  openMenu: (picked: Picked, anchor: MenuAnchor) => void
  closeMenu: () => void
}>((set) => ({
  selected: null,
  menu: null,
  select: (selected) => set({ selected }),
  openMenu: (picked, anchor) => set({ menu: { ...picked, anchor } }),
  closeMenu: () => set({ menu: null }),
}))

/** The hit next to the selected one, step places on (Up is -1, Down 1), or the first hit when none is selected. */
function stepSelection(
  items: BrowseItem[],
  selected: BrowseItem | null,
  step: number,
): BrowseItem | undefined {
  if (items.length === 0) {
    return
  }
  const at = selected ? items.findIndex((i) => hitKey(i) === hitKey(selected)) : -1
  if (at < 0) {
    return items[0]
  }
  return items[Math.min(items.length - 1, Math.max(0, at + step))]
}

export { hitKey, stepSelection, useBrowseSelection }
