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
  dismissed: {},
  nexusUserId: 0,
  nexusName: '',
  nexusPremium: false,
}

export const useSettings = create<Settings>(() => defaults)

export async function initSettings(): Promise<void> {
  const apply = (next: Settings) => useSettings.setState(next)
  Events.On('settings:changed', (event) => {
    apply(event.data)
  })
  apply(await Get())
}
