import { create } from 'zustand'
import type {
  Info,
  Release,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/updatesvc/models.ts'
import {
  Check,
  Info as GetInfo,
  Install,
  Restart,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/updatesvc/service.ts'
import { errorMessage } from '../toasts/report.ts'

export type Phase =
  | 'idle'
  | 'checking'
  | 'current'
  | 'available'
  | 'installing'
  | 'ready'
  | 'restarting'
  | 'error'

// Mortar's own update, held here so the app menu's Check for updates and Settings › Updates share one check.
export const useMortarUpdate = create<{
  info: Info | null
  phase: Phase
  release: Release | null
  error: string
  load: () => Promise<void>
  check: () => Promise<void>
  install: () => Promise<void>
  restart: () => Promise<void>
}>((set, get) => {
  const run = async (
    busy: Phase,
    step: () => Promise<Partial<{ phase: Phase; release: Release | null }>>,
  ) => {
    set({ phase: busy, error: '' })
    try {
      set(await step())
    } catch (e) {
      set({ phase: 'error', error: errorMessage(e) })
    }
  }
  return {
    info: null,
    phase: 'idle',
    release: null,
    error: '',
    load: async () => {
      if (!get().info) {
        set({ info: await GetInfo() })
      }
    },
    check: async () => {
      await get().load()
      const { info, phase } = get()
      // An update already found or staged stays on offer; checking again would hide Install or Restart now.
      if (info?.off || (phase !== 'idle' && phase !== 'current' && phase !== 'error')) {
        return
      }
      await run('checking', async () => {
        const release = await Check()
        if (!release) {
          return { release, phase: 'current' }
        }
        return { release, phase: release.staged ? 'ready' : 'available' }
      })
    },
    install: () =>
      run('installing', async () => {
        await Install()
        return { phase: 'ready' }
      }),
    restart: () =>
      run('restarting', async () => {
        await Restart()
        return { phase: 'restarting' }
      }),
  }
})
