import type { MouseEvent } from 'react'
import { create } from 'zustand'

type TitleMenuId = 'app' | 'game' | 'profile'

export const useOpenTitleMenu = create<{ id: TitleMenuId | null; anchor: HTMLElement | null }>(
  () => ({
    id: null,
    anchor: null,
  }),
)

export const closeTitleMenu = () => useOpenTitleMenu.setState({ id: null, anchor: null })

// One title bar menu is open at a time, like a desktop menu bar: a click on another trigger closes the open menu and
// opens its own, and moving over another trigger while one is open switches to it.
export function useTitleMenu(id: TitleMenuId) {
  const anchor = useOpenTitleMenu((s) => (s.id === id ? s.anchor : null))
  const open = (el: HTMLElement) => useOpenTitleMenu.setState({ id, anchor: el })
  return {
    anchor,
    close: closeTitleMenu,
    trigger: {
      onClick: (e: MouseEvent<HTMLElement>) => (anchor ? closeTitleMenu() : open(e.currentTarget)),
      onMouseEnter: (e: MouseEvent<HTMLElement>) => {
        const { id: current } = useOpenTitleMenu.getState()
        if (current !== null && current !== id) {
          open(e.currentTarget)
        }
      },
    },
  }
}
