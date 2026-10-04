import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type { Status } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loader/models.ts'
import {
  Install,
  InstallVersion,
  Status as LoaderStatus,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loadersvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { errorDetails } from '../toasts/errorKind.ts'
import { errorMessage, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

export const useLoader = create<{
  status: Status | null
  installing: boolean
  steps: string[]
  // Why the last install failed; empty while one runs or after one succeeded.
  error: string
  // The raw cause behind error, for a Details line.
  errorDetail: string
  // An install call is in flight; the loader:state event that sets installing lands after the click, so a second
  // click could otherwise start another.
  pending: boolean
  check: (game: string) => Promise<void>
  install: (game: string, version?: string) => Promise<void>
}>((set, get) => ({
  status: null,
  installing: false,
  pending: false,
  steps: [],
  error: '',
  errorDetail: '',
  check: async (game) => {
    set({ status: null })
    try {
      set({ status: await LoaderStatus(game) })
    } catch {
      // The game may not be installed; there is nothing to offer then.
    }
  },
  // Progress and the outcome arrive as events, so an install Mortar starts by itself shows the same way.
  install: async (game, version) => {
    if (get().pending) {
      return
    }
    set({ pending: true })
    try {
      const status = version ? await InstallVersion(game, version) : await Install(game)
      set({ status })
      useToasts
        .getState()
        .push({ kind: 'success', title: i18n._(msg`SMAPI ${status.version} is installed`) })
    } catch (e) {
      set({ error: errorMessage(e), errorDetail: errorDetails(e) })
      toastError(i18n._(msg`Could not install SMAPI`), e)
    } finally {
      set({ pending: false })
    }
  },
}))

export function initLoader() {
  Events.On('loader:state', (event) => {
    const { game, installing, error } = event.data
    if (installing) {
      useLoader.setState({ installing: true, steps: [], error: '', errorDetail: '' })
      return
    }
    useLoader.setState({
      installing: false,
      error: error ? errorMessage(error) : '',
      errorDetail: error ? errorDetails(error) : '',
    })
    if (!error) {
      useLoader.getState().check(game)
    }
  })
  Events.On('loader:progress', (event) => {
    useLoader.setState((s) => ({ steps: [...s.steps, event.data.step] }))
  })
}
