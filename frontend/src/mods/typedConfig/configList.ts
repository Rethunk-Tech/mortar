import { useEffect } from 'react'
import { create } from 'zustand'
import type {
  ModConfig,
  ModList,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/models.ts'
import { Mods } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/service.ts'
import type { Profile } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useTab } from '../../game/tab.ts'
import { useProfiles } from '../../profiles/store.ts'
import { readStored, writeStored } from '../../shell/useStoredState.ts'
import { reportUnexpected } from '../../toasts/report.ts'

type ConfigShow = 'all' | 'changed' | 'menu' | 'waiting'

type ConfigChip = { kind: 'waiting'; n: number } | { kind: 'changed' } | { kind: 'menu' } | null

const isString = (v: unknown): v is string => typeof v === 'string'

// The row's one chip: edits waiting for the next start first, then a changed file, then the in-game menu.
function chipOf(m: Pick<ModConfig, 'pending' | 'hasMenu' | 'files'>): ConfigChip {
  if (m.pending > 0) {
    return { kind: 'waiting', n: m.pending }
  }
  if ((m.files ?? []).some((f) => f.changed)) {
    return { kind: 'changed' }
  }
  return m.hasMenu ? { kind: 'menu' } : null
}

const matchesShow = (m: ModConfig, show: ConfigShow): boolean => {
  switch (show) {
    case 'changed':
      return (m.files ?? []).some((f) => f.changed)
    case 'menu':
      return m.hasMenu
    case 'waiting':
      return m.pending > 0
    default:
      return true
  }
}

// The mods the Config list shows: those matching the Show choice, then those whose name or id holds the query.
function filterConfigMods(mods: ModConfig[], query: string, show: ConfigShow): ModConfig[] {
  const needle = query.trim().toLowerCase()
  return mods.filter(
    (m) =>
      matchesShow(m, show) &&
      (needle === '' ||
        m.name.toLowerCase().includes(needle) ||
        m.id.toLowerCase().includes(needle)),
  )
}

// A selection is a mod by its id, or a .cfg file nobody owns by its name.
const modSelection = (id: string) => `mod:${id}`
const fileSelection = (name: string) => `file:${name}`

const selectionKey = (profile: string) => `mortar.config.${profile}`

// The Config page's list per profile, with the profile change stamp it was read at, so every screen shares one read.
const useConfigList = create<{
  byProfile: Record<string, { list: ModList; stamp: string } | undefined>
  selected: Record<string, string | undefined>
  // Reads the list unless one is held for this stamp; `force` reads it again (after an edit).
  load: (game: string, profile: string, stamp: string, force?: boolean) => Promise<void>
  select: (profile: string, selection: string) => void
}>((set, get) => ({
  byProfile: {},
  selected: {},
  load: async (game, profile, stamp, force = false) => {
    if (!force && get().byProfile[profile]?.stamp === stamp) {
      return
    }
    const list = await Mods(game, profile)
    set((s) => ({ byProfile: { ...s.byProfile, [profile]: { list, stamp } } }))
  },
  select: (profile, selection) => {
    writeStored(selectionKey(profile), selection)
    set((s) => ({ selected: { ...s.selected, [profile]: selection } }))
  },
}))

// The profile's remembered selection, or the first mod's when none was made or it is gone.
function selectionOf(
  list: ModList | undefined,
  stored: string | undefined,
  profile: string,
): string {
  const want = stored ?? readStored(selectionKey(profile), '', isString)
  const mods = list?.mods ?? []
  const other = list?.other ?? []
  if (
    mods.some((m) => modSelection(m.id) === want) ||
    other.some((f) => fileSelection(f.name) === want)
  ) {
    return want
  }
  const [first] = mods
  if (first) {
    return modSelection(first.id)
  }
  const [file] = other
  return file ? fileSelection(file.name) : ''
}

// Keeps the open profile's list read, so menus can tell at once whether a mod has settings.
function useConfigListLoaded(game: string, profile: Pick<Profile, 'id' | 'updated'>) {
  const load = useConfigList((s) => s.load)
  const stamp = String(profile.updated)
  useEffect(() => {
    load(game, profile.id, stamp).catch(reportUnexpected)
  }, [load, game, profile.id, stamp])
}

// Opens the Config page on the mod.
function openConfigPage(id: string) {
  const { openId } = useProfiles.getState()
  if (openId !== '') {
    useConfigList.getState().select(openId, modSelection(id))
  }
  useTab.getState().setTab('config')
}

const hasConfig = (list: ModList | undefined, id: string): boolean =>
  (list?.mods ?? []).some((m) => m.id === id)

export type { ConfigChip, ConfigShow }
export {
  chipOf,
  fileSelection,
  filterConfigMods,
  hasConfig,
  modSelection,
  openConfigPage,
  selectionOf,
  useConfigList,
  useConfigListLoaded,
}
