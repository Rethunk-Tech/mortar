import { create } from 'zustand'
import type { Mod } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { errorMessage, reportUnexpected } from '../../toasts/report.ts'
import { configApi } from './api.ts'
import type { ConfigFile, ConfigValue } from './types.ts'

const SAVE_DELAY_MS = 500

interface Target {
  game: string
  profile: string
  key: string
  id: string
}

const timers = new Map<string, ReturnType<typeof setTimeout>>()

// The typed editor open on one mod: its config files, the one shown, and a write error per entry.
const useTypedConfig = create<{
  mod: Mod | null
  target: Target | null
  files: ConfigFile[]
  current: string
  errors: Record<string, string>
  loadError: string
  open: (mod: Mod, target: Target) => Promise<void>
  close: () => void
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
    mod: null,
    target: null,
    files: [],
    current: '',
    errors: {},
    loadError: '',
    open: async (mod, target) => {
      set({ mod, target, files: [], current: '', errors: {}, loadError: '' })
      try {
        const files = await configApi.files(target)
        set({ files })
        await get().select(files[0]?.name ?? '')
      } catch (e) {
        set({ loadError: errorMessage(e) })
      }
    },
    close: () => set({ mod: null, target: null }),
    select: async (file) => {
      const { target } = get()
      set({ current: file })
      if (!(target && file)) {
        return
      }
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
