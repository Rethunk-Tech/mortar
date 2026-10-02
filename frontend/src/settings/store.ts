import { create } from 'zustand'
import type { Settings } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import { Get } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { follow } from '../shell/follow.ts'

const defaults: Settings = {
  language: '',
  accent: 'sand',
  background: 'image',
  backgroundImage: '',
  lastGame: '',
  lastProfile: {},
  lastPlayed: {},
  gameFolders: {},
  gameStores: {},
  launcherRoots: {},
  launchersConfirmed: false,
  loaders: {},
  dismissed: {},
  nexusUserId: 0,
  nexusName: '',
  nexusPremium: false,
  nxmHandled: false,
  nxmPrevious: '',
  nxmPreviousName: '',
  nxmAsked: false,
  nxmRedirectOtherGames: null,
  nexusPreferredDownloadServer: '',
  nexusSeenDownloadServers: [],
  checkModUpdatesOnStart: true,
  tellWhenSmapiOut: true,
  askEndorseMods: true,
  keepInTray: false,
  includeBetaReleases: false,
  includePrereleaseModVersions: false,
  checkOnlyEnabledMods: false,
  enableModsWhenInstalled: true,
  backupsKept: 5,
  listColumns: ['on', 'name', 'version', 'author', 'source', 'category', 'status'],
  listSortColumn: 'name',
  listSortDir: 'asc',
  listGroupBy: 'status',
  tipsSeen: [],
  smapiToastAt: '',
  overlayEnabled: false,
  overlayPort: 8123,
  overlayToken: '',
}

export const useSettings = create<Settings>(() => defaults)

export const initSettings = () =>
  follow('settings:changed', Get, (next) => useSettings.setState(next))
