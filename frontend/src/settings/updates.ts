import { msg } from '@lingui/core/macro'
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
import { i18n } from '../i18n/index.ts'
import { errorMessage } from '../toasts/report.ts'
import { BusySummary } from '../../bindings/github.com/Rethunk-AI/mortar/quitservice.ts'

type Phase =
  | 'idle'
  | 'checking'
  | 'current'
  | 'none'
  | 'available'
  | 'installing'
  | 'ready'
  | 'restarting'
  | 'error'

function checkFailure(e: unknown): { phase: Phase; error: string } {
  const raw = errorMessage(e)
  if (raw === 'none') {
    return { phase: 'none', error: '' }
  }
  if (raw === 'unreachable') {
    return {
      phase: 'error',
      error: i18n._(msg`Could not reach GitHub. Check your connection and try again.`),
    }
  }
  return { phase: 'error', error: i18n._(msg`Could not check for updates.`) }
}

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
    fail?: (e: unknown) => { phase: Phase; error: string },
  ) => {
    set({ phase: busy, error: '' })
    try {
      set(await step())
    } catch (e) {
      set(
        fail?.(e) ?? {
          phase: 'error',
          error: i18n._(msg`Could not update: ${errorMessage(e)}`),
        },
      )
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
      if (
        info?.off ||
        (phase !== 'idle' && phase !== 'current' && phase !== 'none' && phase !== 'error')
      ) {
        return
      }
      await run(
        'checking',
        async () => {
          const release = await Check()
          if (!release) {
            return { release, phase: 'current' }
          }
          return { release, phase: release.staged ? 'ready' : 'available' }
        },
        checkFailure,
      )
    },
    install: () => {
      if (get().phase !== 'available') {
        return Promise.resolve()
      }
      return run('installing', async () => {
        await Install()
        return { phase: 'ready' }
      })
    },
    restart: () =>
      run('restarting', async () => {
        const summary = await BusySummary()
        if (summary && !window.confirm(summary)) {
          return { phase: 'ready' }
        }
        await Restart()
        return { phase: 'restarting' }
      }),
  }
})

export const getInitialState = () => useMortarUpdate.getInitialState()

export type { Phase }
