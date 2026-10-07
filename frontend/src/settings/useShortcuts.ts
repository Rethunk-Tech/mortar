import { useEffect } from 'react'
import { useCommandPalette } from '../commandPalette/store.ts'
import { useConsole } from '../console/store.ts'
import { useRenameRequest } from '../game/renameRequest.ts'
import { useSidebarCollapsed } from '../game/sidebarCollapsed.ts'
import { useTab } from '../game/tab.ts'
import { playOpenProfile, playVanillaOpen } from '../launch/playOpen.ts'
import { requestFilterFocus } from '../mods/filterFocus.ts'
import { routeGame, useNav } from '../nav/store.ts'
import { requestFindAllFocus } from '../profiles/findMod.ts'
import { useProfiles } from '../profiles/store.ts'
import { useQueue } from '../queue/store.ts'
import { openImport, openShare } from '../share/store.ts'
import { checkForUpdates } from '../shell/checkForUpdates.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import {
  dialogOpen,
  matchShortcut,
  mergeBindings,
  type ShortcutId,
  shortcutAllowed,
  shortcutCapturing,
} from './shortcuts.ts'
import { useSettings } from './store.ts'

function stepProfile(dir: -1 | 1) {
  const { profiles, openId, open } = useProfiles.getState()
  if (profiles.length === 0) {
    return
  }
  const i = profiles.findIndex((p) => p.id === openId)
  const from = i < 0 ? 0 : i
  const next = profiles[(from + dir + profiles.length) % profiles.length]
  if (next) {
    open(next.id)
  }
}

// Keys the mods list and its rows handle where they are focused. A window-level handler that claimed them would
// cancel Enter, Space and the arrows on every button, tab and menu in the app.
const scopedToMods: ReadonlySet<ShortcutId> = new Set([
  'select-all-mods',
  'mod-up',
  'mod-down',
  'mod-toggle',
  'mod-details',
  'mod-remove',
])

export const isWindowShortcut = (id: ShortcutId) => id !== 'dismiss' && !scopedToMods.has(id)

export function runShortcut(id: ShortcutId) {
  const profiles = useProfiles.getState()
  switch (id) {
    case 'command-palette':
      return useCommandPalette.getState().setOpen(true)
    case 'filter-mods':
      return requestFilterFocus()
    case 'play':
      return playOpenProfile()
    case 'check-updates':
      return checkForUpdates()
    case 'open-settings':
      return useNav.getState().openSettings()
    case 'tab-browse':
      return useTab.getState().setTab('browse')
    case 'tab-load-order':
      return useTab.getState().setTab('load-order')
    case 'tab-mods':
      return useTab.getState().setTab('mods')
    case 'tab-problems':
      return useTab.getState().setTab('problems')
    case 'tab-saves':
      return useTab.getState().setTab('saves')
    case 'tab-config':
      return useTab.getState().setTab('config')
    case 'tab-console':
      return useTab.getState().setTab('console')
    case 'tab-performance':
      return useTab.getState().setTab('performance')
    case 'new-profile':
      return useCommandPalette.getState().setCreating(true)
    case 'duplicate-profile':
      return profiles.duplicate(profiles.openId).catch(reportUnexpected)
    case 'rename-profile':
      return useRenameRequest.getState().request(profiles.openId)
    case 'import':
      return openImport()
    case 'export-profile':
      return openShare(profiles.openId)
    case 'back': {
      const nav = useNav.getState()
      if (nav.route.name === 'settings') {
        nav.closeSettings()
      } else if (nav.route.name === 'profiles') {
        nav.closeProfiles()
      } else if (nav.route.name === 'game-settings') {
        nav.closeGameSettings()
      }
      return
    }
    case 'help':
      return useConsole.getState().setHelping(true)
    case 'downloads': {
      const queue = useQueue.getState()
      return queue.setOpen(!queue.open)
    }
    case 'notifications':
      return useToasts.getState().setHistoryOpen(true)
    case 'previous-profile':
      return stepProfile(-1)
    case 'next-profile':
      return stepProfile(1)
    case 'collapse-sidebar':
      return useSidebarCollapsed.getState().toggle()
    case 'find-all-mods': {
      if (!routeGame(useNav.getState().route)) {
        return
      }
      useNav.getState().openProfiles()
      return requestFindAllFocus()
    }
    case 'vanilla-play':
      return playVanillaOpen()
    default:
      return
  }
}

export function useAppShortcuts() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (shortcutCapturing()) {
        return
      }
      const id = matchShortcut(e, mergeBindings(useSettings.getState().shortcuts))
      const typing = e.target instanceof HTMLElement ? e.target : null
      if (!(id && shortcutAllowed(id, typing, dialogOpen()))) {
        return
      }
      if (!isWindowShortcut(id)) {
        return
      }
      e.preventDefault()
      runShortcut(id)
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [])
}
