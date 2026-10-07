import { create } from 'zustand'
import { errorMessage, reportUnexpected } from '../../toasts/report.ts'
import { configApi } from './api.ts'
import { useConfigList } from './configList.ts'
import type { ConfigFile, ConfigValue } from './types.ts'

const SAVE_DELAY_MS = 500

interface Target {
  game: string
  profile: string
  key: string
  id: string
}

const timers = new Map<string, ReturnType<typeof setTimeout>>()

// The typed editor open on one mod, or on the .cfg files no mod owns: its config files, the one shown, and a write error per entry.
const useTypedConfig = create<{
  target: Target | null
  files: ConfigFile[]
  current: string
  errors: Record<string, string>
  loadError: string
  open: (target: Target, file?: string) => Promise<void>
  select: (file: string) => Promise<void>
  set: (section: string, key: string, value: ConfigValue) => void
  reset: (section: string, key: string) => void
  resetAll: () => Promise<void>
}>((set, get) => {
  const edit = (
    section: string,
    key: string,
    fn: (v: ConfigValue, d: ConfigValue) => ConfigValue,
  ) =>
    set((s) => ({
      files: s.files.map((f) =>
        f.name === s.current
          ? {
              ...f,
              sections: f.sections.map((sec) =>
                sec.name === section
                  ? {
                      ...sec,
                      entries: sec.entries.map((e) =>
                        e.key === key ? { ...e, value: fn(e.value, e.default) } : e,
                      ),
                    }
                  : sec,
              ),
            }
          : f,
      ),
    }))
  const save = async (slot: string, run: () => Promise<void>) => {
    try {
      await run()
    } catch (e) {
      set((s) => ({ errors: { ...s.errors, [slot]: errorMessage(e) } }))
      return
    }
    set((s) => ({ errors: { ...s.errors, [slot]: '' } }))
    // The list's Changed and waiting chips follow the edit.
    const { target } = get()
    if (target) {
      useConfigList.getState().load(target.game, target.profile, '', true).catch(reportUnexpected)
    }
    // An in-game menu edit turns pending, or stops being, only as the profile records it.
    const { files, current } = get()
    if (files.find((f) => f.name === current)?.format === 'gmcm') {
      await get().select(current)
    }
  }
  const write = (section: string, key: string, run: () => Promise<void>) => {
    const slot = `${get().current}/${section}/${key}`
    clearTimeout(timers.get(slot))
    timers.set(
      slot,
      setTimeout(() => {
        timers.delete(slot)
        save(slot, run).catch(reportUnexpected)
      }, SAVE_DELAY_MS),
    )
  }
  return {
    target: null,
    files: [],
    current: '',
    errors: {},
    loadError: '',
    open: async (target, file) => {
      set({ target, files: [], current: '', errors: {}, loadError: '' })
      try {
        const files = await configApi.files(target)
        set({ files })
        await get().select(files.find((f) => f.name === file)?.name ?? files[0]?.name ?? '')
      } catch (e) {
        set({ loadError: errorMessage(e) })
      }
    },
    select: async (file) => {
      const { target, files } = get()
      // A file the opened mod does not have is never asked for: a pane that renders before open() clears the previous
      // mod's state would otherwise request that mod's file name.
      if (!(target && files.some((f) => f.name === file))) {
        return
      }
      set({ current: file })
      try {
        const sections = await configApi.schema(target, file)
        set((s) => ({ files: s.files.map((f) => (f.name === file ? { ...f, sections } : f)) }))
      } catch (e) {
        set({ loadError: errorMessage(e) })
      }
    },
    set: (section, key, value) => {
      const { target, current } = get()
      edit(section, key, () => value)
      if (target) {
        write(section, key, () =>
          configApi.set(target, { file: current, section, entry: key }, value),
        )
      }
    },
    reset: (section, key) => {
      const { target, current } = get()
      edit(section, key, (_v, d) => d)
      if (target) {
        write(section, key, () => configApi.reset(target, { file: current, section, entry: key }))
      }
    },
    resetAll: async () => {
      const { target, current } = get()
      if (!target) {
        return
      }
      try {
        await configApi.resetAll(target, current)
        await get().select(current)
      } catch (e) {
        reportUnexpected(e)
      }
    },
  }
})

interface EntryAt {
  file: string
  section: string
  entry: string
}

export type { EntryAt, Target }
export { useTypedConfig }
