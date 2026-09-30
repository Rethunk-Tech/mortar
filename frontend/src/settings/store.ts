import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type { Settings } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import { Get } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'

const defaults: Settings = {
  accent: 'sand',
  translucent: true,
  lastGame: '',
  lastProfile: {},
  gameFolders: {},
  loaders: {},
}

export const useSettings = create<Settings>(() => defaults)

const apply = (next: Settings) => useSettings.setState(next)

export async function initSettings(): Promise<void> {
  Events.On('settings:changed', (event) => {
    apply(event.data)
  })
  apply(await Get())
}
