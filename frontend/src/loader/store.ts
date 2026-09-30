import { t } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type { Status } from '../../bindings/github.com/Rethunk-AI/mortar/internal/loader/models.ts'
import {
  Install,
  Status as LoaderStatus,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/loadersvc/service.ts'
import { useToasts } from '../toasts/store.ts'

export const useLoader = create<{
  status: Status | null
  installing: boolean
  steps: string[]
  check: (game: string) => Promise<void>
  install: (game: string) => Promise<void>
}>((set) => ({
  status: null,
  installing: false,
  steps: [],
  check: async (game) => {
    set({ status: null })
    try {
      set({ status: await LoaderStatus(game) })
    } catch {
      // The game may not be installed; there is nothing to offer then.
    }
  },
  install: async (game) => {
    set({ installing: true, steps: [] })
    const off = Events.On('loader:progress', (event) => {
      if (event.data.game === game) {
        set((s) => ({ steps: [...s.steps, event.data.step] }))
      }
    })
    try {
      const status = await Install(game)
      set({ status })
      useToasts.getState().push({ kind: 'success', title: t`SMAPI ${status.version} is installed` })
    } catch (e) {
      useToasts
        .getState()
        .push({ kind: 'error', title: t`Could not install SMAPI`, body: String(e) })
    } finally {
      off()
      set({ installing: false })
    }
  },
}))
