import type { MouseEvent } from 'react'

export interface GameMenuAt {
  id: string
  top: number
  left: number
}

// Right-click on a row, or the menu key while a row holds focus, opens that game's menu.
export function gameMenuAt(e: MouseEvent<HTMLElement>): GameMenuAt | null {
  const tile = (e.target as HTMLElement).closest<HTMLElement>('[data-tile]')
  const id = tile?.dataset.game
  if (!(tile && id)) {
    return null
  }
  if (e.clientX !== 0 || e.clientY !== 0) {
    return { id, top: e.clientY, left: e.clientX }
  }
  const box = (document.activeElement ?? tile).getBoundingClientRect()
  return { id, top: box.bottom, left: box.left }
}
