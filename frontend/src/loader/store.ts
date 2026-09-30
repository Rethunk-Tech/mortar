import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type { Status } from '../../bindings/github.com/Rethunk-AI/mortar/internal/loader/models.ts'
import {
  Install,
  Status as LoaderStatus,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/loadersvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { useToasts } from '../toasts/store.ts'

export const useLoader = create<{
  status: Status | null
  installing: boolean
  steps: string[]
  // Why the last install failed; empty while one runs or after one succeeded.
  error: string
  check: (game: string) => Promise<void>
  install: (game: string) => Promise<void>
}>((set) => ({
  status: null,
  installing: false,
  steps: [],
  error: '',
  check: async (game) => {
    set({ status: null })
    try {
      set({ status: await LoaderStatus(game) })
    } catch {
      // The game may not be installed; there is nothing to offer then.
    }
  },
  // Progress and the outcome arrive as events, so an install Mortar starts by itself shows the same way.
  install: async (game) => {
    try {
      const status = await Install(game)
      set({ status })
      useToasts
        .getState()
        .push({ kind: 'success', title: i18n._(msg`SMAPI ${status.version} is installed`) })
    } catch (e) {
      useToasts
        .getState()
        .push({ kind: 'error', title: i18n._(msg`Could not install SMAPI`), body: String(e) })
    }
  },
}))

export function initLoader() {
  Events.On('loader:state', (event) => {
    const { game, installing, error } = event.data
    if (installing) {
      useLoader.setState({ installing: true, steps: [], error: '' })
      return
    }
    useLoader.setState({ installing: false, error })
    if (!error) {
      useLoader.getState().check(game)
    }
  })
  Events.On('loader:progress', (event) => {
    useLoader.setState((s) => ({ steps: [...s.steps, event.data.step] }))
  })
}
