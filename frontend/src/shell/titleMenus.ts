import type { MouseEvent } from 'react'
import { create } from 'zustand'

type TitleMenuId = 'app' | 'game' | 'profile'

export const useOpenTitleMenu = create<{
  id: TitleMenuId | null
  anchor: HTMLElement | null
  // Opened by moving over the trigger, so the click that follows on it keeps the menu open.
  byHover: boolean
}>(() => ({ id: null, anchor: null, byHover: false }))

export const closeTitleMenu = () =>
  useOpenTitleMenu.setState({ id: null, anchor: null, byHover: false })

// One title bar menu is open at a time, like a desktop menu bar: a click on another trigger closes the open menu and
// opens its own, and moving over another trigger while one is open switches to it.
export function titleMenuTrigger(id: TitleMenuId) {
  const open = (el: HTMLElement, byHover: boolean) =>
    useOpenTitleMenu.setState({ id, anchor: el, byHover })
  return {
    onClick: (e: MouseEvent<HTMLElement>) => {
      const { id: current, byHover } = useOpenTitleMenu.getState()
      if (current === id && !byHover) {
        closeTitleMenu()
      } else {
        open(e.currentTarget, false)
      }
    },
    onMouseEnter: (e: MouseEvent<HTMLElement>) => {
      const { id: current } = useOpenTitleMenu.getState()
      if (current !== null && current !== id) {
        open(e.currentTarget, true)
      }
    },
  }
}

export function useTitleMenu(id: TitleMenuId) {
  const anchor = useOpenTitleMenu((s) => (s.id === id ? s.anchor : null))
  return { anchor, close: closeTitleMenu, trigger: titleMenuTrigger(id) }
}
