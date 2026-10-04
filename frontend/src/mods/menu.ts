import { Browser } from '@wailsio/runtime'
import type { KeyboardEvent, MouseEvent } from 'react'
import { create } from 'zustand'
import type {
  Entry,
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  History,
  Revert,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { undoRevertTarget } from '../toasts/undo.ts'
import { entryOf, modId, updateFor } from './lookup.ts'
import { hostOf, type MenuState } from './modActions.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'

const NEXUS_KEY = /^nexus-(\d+)-(\d+)$/

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

export const ICON_SIZE = 16

export const openPage = (url: string) => Browser.OpenURL(url).catch(reportUnexpected)

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

export function extraFileLabel(
  entry: Entry | undefined,
  extraKey: string,
  files: { fileId: number; fileName: string; name: string; version: string }[] | null | undefined,
): string {
  if (!entry) {
    return extraKey
  }
  const match = NEXUS_KEY.exec(extraKey)
  const fileId = match ? Number(match[2]) : 0
  const file = files?.find((candidate) => candidate.fileId === fileId)
  if (file) {
    const title = file.fileName || file.name
    return file.version ? `${title} (${file.version})` : title
  }
  const prefix = `${extraKey.replaceAll('\\', '/')}/`
  const mods = (entry.mods ?? []).filter((mod) => {
    const folder = (mod.folder ?? '').replaceAll('\\', '/')
    return folder === extraKey || folder.startsWith(prefix)
  })
  if (mods.length > 0) {
    return mods.map((mod) => (mod.version ? `${mod.name} (${mod.version})` : mod.name)).join(', ')
  }
  return extraKey
}

export function entryFileLabel(entry: Entry): string {
  const name = entry.source?.name || entry.mods?.[0]?.name || entry.key
  const version = entry.source?.version || entry.mods?.[0]?.version || ''
  return version ? `${name} (${version})` : name
}

export function samePageSiblings(profile: Profile | undefined, entry: Entry | undefined): Entry[] {
  const pageId = entry?.source?.kind === 'nexus' ? (entry.source.modId ?? 0) : 0
  if (!(profile && entry && pageId > 0)) {
    return []
  }
  return (profile.entries ?? []).filter((other) => {
    if (
      other.key === entry.key ||
      other.source?.kind !== 'nexus' ||
      (other.source.modId ?? 0) !== pageId
    ) {
      return false
    }
    return (other.extraStoreKeys ?? []).length === 0
  })
}

export async function applyWithUndo(
  gameId: string,
  profileId: string,
  change: () => Promise<Profile>,
  toast: (undo: () => unknown) => void,
): Promise<void> {
  const beforeId = undoRevertTarget((await History(gameId, profileId)) ?? [])
  useProfiles.getState().replace(await change())
  toast(async () => {
    if (!beforeId) {
      return
    }
    useProfiles.getState().replace(await Revert(gameId, profileId, beforeId))
  })
}
