import { create } from 'zustand'
import type {
  AssetConflict,
  SettingHint,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/framework/models.ts'
import type {
  Duplicate,
  Result,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useSettings } from '../settings/store.ts'
import { dependentsOf } from './dependents.ts'
import { requestPinWithReason } from './pinReasonStore.ts'
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
  skipSourceMany,
  skipVersion,
  skipVersionMany,
  setUpdateChannel as writeUpdateChannel,
} from './storeEntries.ts'
import { loadModProblems, loadMods, showUpdatesView } from './storeLoad.ts'
import { problemActions } from './storeProblems.ts'
import {
  removingOf,
  storedView,
  storeFilter,
  storeTags,
  type View,
  viewActions,
} from './storeView.ts'
import type { ModFilter } from './Toolbar.tsx'

export const useMods = create<{
  mods: Mod[]
  loaded: boolean
  // modsFor is the profile mods belongs to, so returning to that profile's tab shows them while they refresh.
  modsFor: string
  queries: Record<string, string>
  filters: Record<string, ModFilter | undefined>
  tagFilters: Record<string, string[] | undefined>
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
  setUpdateChannel: (mod: Mod, channel: string) => Promise<void>
  setSkipSource: (mod: Mod, source: string, skip: boolean) => Promise<void>
  setSkipSourceMany: (groups: ReadonlyMap<string, Mod[]>, skip: boolean) => Promise<void>
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
  dismissAbandoned: (id: string) => Promise<void>
  dismissListed: (id: string) => Promise<void>
  dismissSetting: (setting: SettingHint) => Promise<void>
  setConfigValue: (
    setting: Pick<SettingHint, 'key' | 'id' | 'field' | 'name'>,
    value: string,
  ) => Promise<void>
  showUpdates: () => void
  setQuery: (profileId: string, query: string) => void
  setFilter: (profileId: string, filter: ModFilter) => void
  setTagFilter: (profileId: string, tags: string[]) => void
}>((set, get) => ({
  mods: [],
  loaded: false,
  modsFor: '',
  queries: {},
  filters: {},
  tagFilters: {},
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
  setFilter: (profileId, filter) => {
    storeFilter(profileId, filter)
    set((s) => ({ filters: { ...s.filters, [profileId]: filter } }))
  },
  setTagFilter: (profileId, tags) => {
    storeTags(profileId, tags)
    set((s) => ({ tagFilters: { ...s.tagFilters, [profileId]: tags } }))
  },
  loadProblems: () => loadModProblems(set, get),
  setEnabled: (mod, enabled) => setEnabledAction(set, get, mod, enabled),
  setEnabledMany: (mods, enabled) => enableMany(set, get, mods, enabled),
  setPinned: (mod, pinned) => {
    const want = requestPinWithReason([mod], pinned)
    return want.unpinned ? pinMod(mod, false) : Promise.resolve()
  },
  setPinnedMany: (mods, pinned) => {
    const want = requestPinWithReason(mods, pinned)
    return want.unpinned ? pinMany(mods, false) : Promise.resolve()
  },
  setSkipVersion: (mod, version) => skipVersion(mod, version),
  setSkipVersionMany: (mods) => skipVersionMany(mods),
  setUpdateChannel: (mod, channel) => writeUpdateChannel(mod, channel),
  setSkipSource: (mod, source, skip) => skipSource(mod, source, skip),
  setSkipSourceMany: (groups, skip) => skipSourceMany(groups, skip),
  setCategoryMany: (mods, category) => setCategoryMany(mods, category),
  setTagMany: (mods, tag, add) => setTagMany(mods, tag, add),
  setNoteTags: (mod, note, tags) => setEntryNoteTags(mod, note, tags),
  askRemove: (mod) => {
    const list = removingOf(mod)
    if (list.length === 0) {
      set({ removing: [] })
      return
    }
    if (
      useSettings.getState().confirmRemovals === false &&
      dependentsOf(get().mods, list).length === 0
    ) {
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
