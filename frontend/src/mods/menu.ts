import { Browser } from '@wailsio/runtime'
import type { KeyboardEvent, MouseEvent } from 'react'
import { create } from 'zustand'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { modId } from './lookup.ts'
import { hostOf, type MenuState } from './modActions.ts'
import { useMods } from './store.ts'

const isMenuKey = (e: { key: string; shiftKey: boolean }) =>
  e.key === 'ContextMenu' || (e.shiftKey && e.key === 'F10')

export function useMenuState(mod: Mod, removable: boolean): MenuState {
  const host = useMods((s) => hostOf(s.pages[modId(mod)]))
  return { enabled: mod.enabled, host, removable }
}

export const openPage = (url: string) => Browser.OpenURL(url)

export type MenuAnchor = { el: HTMLElement } | { top: number; left: number }

// The one right-click menu of the mods screen: which mod it is for and where it opens.
export const useContextMenu = create<{
  target: { mod: Mod; removable: boolean; anchor: MenuAnchor } | null
  open: (mod: Mod, removable: boolean, anchor: MenuAnchor) => void
  close: () => void
}>((set) => ({
  target: null,
  open: (mod, removable, anchor) => set({ target: { mod, removable, anchor } }),
  close: () => set({ target: null }),
}))

// The props that make a right-click, the Menu key or Shift+F10 on the element open the mod's menu.
export function contextMenuProps(mod: Mod, removable: boolean) {
  const { open } = useContextMenu.getState()
  return {
    onContextMenu: (e: MouseEvent<HTMLElement>) => {
      e.preventDefault()
      open(mod, removable, { top: e.clientY, left: e.clientX })
    },
    onKeyDown: (e: KeyboardEvent<HTMLElement>) => {
      if (isMenuKey(e)) {
        e.preventDefault()
        open(mod, removable, { el: e.currentTarget })
      }
    },
  }
}
