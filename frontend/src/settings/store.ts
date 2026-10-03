import { create } from 'zustand'
import type { Settings } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import {
  CorruptSettingsPath,
  Get,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { follow } from '../shell/follow.ts'
import { useToasts } from '../toasts/store.ts'

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
  lanSharing: false,
  lanPort: 47_630,
  lanAddresses: [],
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
  shortcuts: {},
  onPlay: 'stay',
  backupBeforePlay: 'changed',
  launchBackupsKept: 5,
  updateModsBeforePlayDefault: false,
  runsKept: 20,
  consoleLogCap: 20000,
  parallelDownloads: 3,
  updateCheckIntervalMinutes: 60,
  notifyModUpdates: false,
  keepDownloadArchives: false,
  storeRetentionDays: 30,
  nxmDefaultProfile: '',
  defaultModsView: 'grid',
  confirmRemovals: true,
  cosmeticConflicts: 'collapsed',
  backgroundBadgeChecks: true,
  startScreen: 'last',
  dates: 'relative',
  trashRetentionDays: 30,
  historyEventsKept: 200,
  notifyDownloadFinished: true,
  notifyDownloadFailed: true,
  notifyRunCrashed: true,
}

export const useSettings = create<Settings>(() => defaults)

export const initSettings = async () => {
  await follow('settings:changed', Get, (next) => useSettings.setState(next))
  const path = await CorruptSettingsPath()
  if (path) {
    useToasts.getState().push({
      kind: 'warning',
      title: 'Settings could not be read',
      body: `A copy was kept at ${path}`,
    })
  }
}
