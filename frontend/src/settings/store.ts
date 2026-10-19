import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type { Settings } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import {
  CorruptSettingsPath,
  Get,
  ShowCorruptSettings,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { i18n } from '../i18n/index.ts'
import { follow } from '../shell/follow.ts'
import { keepFields } from '../shell/sameValue.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { loadPrefSpecs } from './prefSpecs.ts'

const defaults: Settings = {
  language: '',
  accent: 'sand',
  font: '',
  background: 'image',
  backgroundImage: '',
  lastGame: '',
  hiddenGames: [],
  lastProfile: {},
  lastPlayed: {},
  gameFolders: {},
  gameStores: {},
  launcherRoots: {},
  launchersConfirmed: false,
  loaders: {},
  dismissed: {},
  thunderstoreHandleLinks: null,
  curseforgeHandleLinks: null,
  vsmoddbHandleLinks: null,
  nexusUserId: 0,
  nexusName: '',
  itchName: '',
  nexusPremium: false,
  nxmHandled: false,
  nxmPreviousHandlers: {},
  nxmPreviousName: '',
  nxmAsked: false,
  nxmRedirectOtherGames: null,
  nexusPreferredDownloadServer: '',
  nexusSeenDownloadServers: [],
  checkModUpdatesOnStart: true,
  smapiBuilds: 'show',
  loaderPrefs: {},
  showSmapiConsole: true,
  askEndorseMods: false,
  keepInTray: true,
  lanSharing: true,
  lanPort: 0,
  lanAllowAnyAddress: false,
  lanStarted: false,
  lanAddresses: [],
  includeBetaReleases: true,
  includePrereleaseModVersions: false,
  checkOnlyEnabledMods: false,
  enableModsWhenInstalled: true,
  listColumns: ['on', 'name', 'version', 'category', 'installed'],
  listSortColumn: 'name',
  listSortDir: 'asc',
  listGroupBy: 'status',
  tipsSeen: [],
  overlayEnabled: false,
  overlayPort: 8123,
  overlayToken: '',
  shortcuts: {},
  onPlay: 'stay',
  games: {},
  parallelDownloads: 3,
  downloadSpeedLimit: 0,
  updateCheckIntervalMinutes: 180,
  notifyModUpdates: true,
  updateDigest: 'daily',
  lastModUpdateDigest: [],
  lastModUpdateDigestAt: '',
  keepDownloadArchives: false,
  storeRetentionDays: 30,
  defaultModsView: 'grid',
  implicitModsView: 'nested',
  confirmRemovals: true,
  backgroundBadgeChecks: true,
  startScreen: 'last',
  antivirus: 'automatic',
  dates: 'relative',
  trashRetentionDays: 30,
  historyEventsKept: 200,
  notifyDownloadFinished: false,
  notifyDownloadFailed: true,
  notifyRunCrashed: true,
  desktopDownloadFinished: false,
  desktopDownloadFailed: true,
  desktopRunCrashed: true,
  desktopModUpdates: false,
  density: 'comfortable',
  theme: 'dark',
  gridCardSize: 'medium',
  showAuthorOnCards: true,
  browseCardStats: 'popularity,downloads',
  reduceMotion: 'system',
  growGameRows: true,
  profileHero: 'full',
  reuseFomodChoices: true,
  driftChecks: true,
  autoInstallMortarUpdates: true,
  autoTrackNexus: false,
  lanName: '',
  lanAutoAcceptPaired: false,
  downloadFolder: '',
  profileOrder: 'manual',
  autoRetryDownloads: 'off',
  thunderstoreHost: 'auto',
  sidebarBadges: 'problemsAndUpdates',
  shareIncludeDisabledMods: true,
  shareIncludeFomodChoices: true,
  shareIncludeNotes: true,
  shareIncludeConfigFiles: true,
  shareIncludeProblemChoices: true,
  shareIncludeLibraries: true,
  verifyNexusMD5: true,
  showAdultContent: false,
  launchAtLogin: false,
  startMinimised: false,
  rememberWindow: true,
  extensionConnection: 'allow',
} as Settings

export const useSettings = create<Settings>(() => defaults)

export const initSettings = async () => {
  await loadPrefSpecs()
  // Unchanged fields, and unchanged per-game blocks, keep their objects, so a component that selects one (the last-played
  // map, a game's column list) does not redraw when another setting is saved.
  await follow('settings:changed', Get, (next) =>
    useSettings.setState((prev) => {
      const kept = keepFields(prev, next)
      return kept.games === prev.games || !(prev.games && kept.games)
        ? kept
        : { ...kept, games: keepFields(prev.games, kept.games) }
    }),
  )
  const path = await CorruptSettingsPath()
  if (path) {
    useToasts.getState().push({
      kind: 'warning',
      title: i18n._(msg`Your settings could not be read`),
      body: i18n._(
        msg`Mortar started with default settings. The damaged file was kept at ${path}.`,
      ),
      action: {
        label: i18n._(msg`Show file`),
        run: () => ShowCorruptSettings().catch(reportUnexpected),
      },
    })
  }
}
