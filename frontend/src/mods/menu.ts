import { Browser } from '@wailsio/runtime'
import type { KeyboardEvent, MouseEvent } from 'react'
import { create } from 'zustand'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { entryOf, modId, updateFor } from './lookup.ts'
import { hostOf, type MenuState } from './modActions.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'

const isMenuKey = (e: { key: string; shiftKey: boolean }) =>
  e.key === 'ContextMenu' || (e.shiftKey && e.key === 'F10')

export function useMenuState(mod: Mod): MenuState {
  const host = useMods((s) => hostOf(s.pages[modId(mod)]))
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === s.openId))
  const entry = entryOf(profile, mod.key)
  const update = useUpdates((s) => updateFor(s.updates, mod, profile))
  return {
    enabled: mod.enabled,
    host,
    pinned: Boolean(entry?.pinned),
    skipVersion: entry?.skipVersion ?? '',
    hasUpdate: Boolean(update),
  }
}

export const openPage = (url: string) => Browser.OpenURL(url)

export type MenuAnchor = { el: HTMLElement } | { top: number; left: number }

// The one right-click menu of the mods screen: which mod it is for and where it opens.
export const useContextMenu = create<{
  target: { mod: Mod; anchor: MenuAnchor } | null
  open: (mod: Mod, anchor: MenuAnchor) => void
  close: () => void
}>((set) => ({
  target: null,
  open: (mod, anchor) => set({ target: { mod, anchor } }),
  close: () => set({ target: null }),
}))

// The props that make a right-click, the Menu key or Shift+F10 on the element open the mod's menu.
export function contextMenuProps(mod: Mod) {
  const { open } = useContextMenu.getState()
  return {
    onContextMenu: (e: MouseEvent<HTMLElement>) => {
      e.preventDefault()
      open(mod, { top: e.clientY, left: e.clientX })
    },
    onKeyDown: (e: KeyboardEvent<HTMLElement>) => {
      if (isMenuKey(e)) {
        e.preventDefault()
        open(mod, { el: e.currentTarget })
      }
    },
  }
}
