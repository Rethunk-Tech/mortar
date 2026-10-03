import { create } from 'zustand'
import type {
  AssetConflict,
  Duplicate,
  Result,
  SettingHint,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useSettings } from '../settings/store.ts'
import {
  dropMod,
  dropMods,
  enableMany,
  openModConfig,
  pinMany,
  pinMod,
  setCategoryMany,
  setEnabledAction,
  setEntryNoteTags,
  setTagMany,
  showModFiles,
  skipSource,
  skipVersion,
  skipVersionMany,
} from './storeEntries.ts'
import { loadModProblems, loadMods, showUpdatesView } from './storeLoad.ts'
import { problemActions } from './storeProblems.ts'
import { removingOf, storedView, type View, viewActions } from './storeView.ts'

export const useMods = create<{
  mods: Mod[]
  loaded: boolean
  // modsFor is the profile mods belongs to, so returning to that profile's tab shows them while they refresh.
  modsFor: string
  queries: Record<string, string>
  loadError: string
  pages: Record<string, string | undefined>
  view: View
  removing: Mod[]
  problems: Result | null
  // problemsFor is the profile problems belongs to, so another profile's result is never shown as this one's.
  problemsFor: string
  resolving: Duplicate | null
  setView: (view: View) => void
  load: () => Promise<void>
  loadProblems: () => Promise<void>
  setEnabled: (mod: Mod, enabled: boolean) => Promise<void>
  setEnabledMany: (mods: Mod[], enabled: boolean) => Promise<void>
  setPinned: (mod: Mod, pinned: boolean) => Promise<void>
  setPinnedMany: (mods: Mod[], pinned: boolean) => Promise<void>
  setSkipVersion: (mod: Mod, version: string) => Promise<void>
  setSkipVersionMany: (mods: Mod[]) => Promise<void>
  setSkipSource: (mod: Mod, source: string, skip: boolean) => Promise<void>
  setCategoryMany: (mods: Mod[], category: string) => Promise<void>
  setTagMany: (mods: Mod[], tag: string, add: boolean) => Promise<void>
  setNoteTags: (mod: Mod, note: string, tags: string[]) => Promise<void>
  askRemove: (mod: Mod | readonly Mod[] | null) => void
  remove: (mod: Mod) => Promise<void>
  removeMany: (mods: Mod[]) => Promise<void>
  showFiles: (mod: Mod) => Promise<void>
  openConfig: (mod: Mod) => Promise<void>
  resolve: (dup: Duplicate | null) => void
  keepCopy: (dup: Duplicate, keepKey: string) => Promise<void>
  dismissAsset: (conflict: AssetConflict) => Promise<void>
  restoreDismissed: (token: string) => Promise<void>
  dismissAbandoned: (uniqueId: string) => Promise<void>
  dismissListed: (uniqueId: string) => Promise<void>
  dismissSetting: (setting: SettingHint) => Promise<void>
  setConfigValue: (
    setting: Pick<SettingHint, 'key' | 'uniqueId' | 'field' | 'name'>,
    value: string,
  ) => Promise<void>
  showUpdates: () => void
  setQuery: (profileId: string, query: string) => void
}>((set, get) => ({
  mods: [],
  loaded: false,
  modsFor: '',
  queries: {},
  loadError: '',
  pages: {},
  view: storedView(),
  removing: [],
  problems: null,
  problemsFor: '',
  resolving: null,
  ...viewActions(set),
  load: () => loadMods(set, get),
  setQuery: (profileId, query) => set((s) => ({ queries: { ...s.queries, [profileId]: query } })),
  loadProblems: () => loadModProblems(set, get),
  setEnabled: (mod, enabled) => setEnabledAction(set, get, mod, enabled),
  setEnabledMany: (mods, enabled) => enableMany(set, get, mods, enabled),
  setPinned: (mod, pinned) => pinMod(mod, pinned),
  setPinnedMany: (mods, pinned) => pinMany(mods, pinned),
  setSkipVersion: (mod, version) => skipVersion(mod, version),
  setSkipVersionMany: (mods) => skipVersionMany(mods),
  setSkipSource: (mod, source, skip) => skipSource(mod, source, skip),
  setCategoryMany: (mods, category) => setCategoryMany(mods, category),
  setTagMany: (mods, tag, add) => setTagMany(mods, tag, add),
  setNoteTags: (mod, note, tags) => setEntryNoteTags(mod, note, tags),
  askRemove: (mod) => {
    const list = removingOf(mod)
    if (list.length === 0) {
      set({ removing: [] })
      return
    }
    if (useSettings.getState().confirmRemovals === false) {
      dropMods(get, list).catch(() => undefined)
      return
    }
    set({ removing: list })
  },
  remove: (mod) => dropMod(get, mod),
  removeMany: (mods) => dropMods(get, mods),
  showFiles: (mod) => showModFiles(mod),
  openConfig: (mod) => openModConfig(mod),
  ...problemActions(set, get),
  showUpdates: () => showUpdatesView(),
}))

export type { View } from './storeView.ts'
