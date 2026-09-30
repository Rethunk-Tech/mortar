import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type { Settings } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import { Get } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'

const defaults: Settings = {
  accent: 'sand',
  background: 'image',
  backgroundImage: '',
  lastGame: '',
  lastProfile: {},
  gameFolders: {},
  loaders: {},
}

export const useSettings = create<Settings>(() => defaults)

// The window type is fixed when the window is created, so a switch to or from solid needs a restart.
export const useLaunchSolid = create<{ solid: boolean }>(() => ({ solid: false }))

export async function initSettings(): Promise<void> {
  const apply = (next: Settings) => useSettings.setState(next)
  Events.On('settings:changed', (event) => {
    apply(event.data)
  })
  const loaded = await Get()
  useLaunchSolid.setState({ solid: loaded.background === 'solid' })
  apply(loaded)
}
