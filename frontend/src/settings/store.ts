import { create } from 'zustand'
import type { Settings } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import { Get } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { follow } from '../shell/follow.ts'

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
  nxmHandled: false,
  nxmPrevious: '',
  nxmAsked: false,
  backupsKept: 5,
}

export const useSettings = create<Settings>(() => defaults)

export const initSettings = () =>
  follow('settings:changed', Get, (next) => useSettings.setState(next))
