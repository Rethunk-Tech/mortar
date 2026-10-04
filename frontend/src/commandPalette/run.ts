import { type TabId, useTab } from '../game/tab.ts'
import { openDownloadsDialog } from '../install/downloadsDialog.ts'
import { playOpenProfile } from '../launch/playOpen.ts'
import { useMods } from '../mods/store.ts'
import { useUpdates } from '../mods/updates.ts'
import { type GameId, type SettingsSection, useNav } from '../nav/store.ts'
import { openModInProfile } from '../profiles/findMod.ts'
import { useRecentChanges } from '../profiles/recentChanges.ts'
import { useProfiles } from '../profiles/store.ts'
import { useQueue } from '../queue/store.ts'
import { SHORTCUTS, type ShortcutId } from '../settings/shortcuts.ts'
import { runShortcut } from '../settings/useShortcuts.ts'
import { openImport, openShare } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { startCrashBisectFromPalette } from './crashBisect.ts'
import { useCommandPalette } from './store.ts'

const sections = new Set<SettingsSection>([
  'general',
  'appearance',
  'mods',
  'downloads',
  'nexus',
  'updates',
  'notifications',
  'storage',
  'launchers',
  'shortcuts',
  'about',
])
const TAB_PREFIX_LENGTH = 4

function leaveShellPages(): void {
  const nav = useNav.getState()
  nav.closeSettings()
  nav.closeProfiles()
  nav.closeGameSettings()
}

function gameId(): GameId | null {
  const id = useProfiles.getState().game?.id
  return id === 'stardew' ? id : null
}

function openProfile(id: string): void {
  const game = gameId()
  if (game) {
    useNav.getState().openGame(game)
  }
  useProfiles.getState().open(id)
  leaveShellPages()
}

function runAction(id: string): void {
  if (id.startsWith('tab:')) {
    useTab.getState().setTab(id.slice(TAB_PREFIX_LENGTH) as TabId)
    return
  }
  if (id.startsWith('configure-mod:')) {
    const rest = id.slice('configure-mod:'.length)
    const cut = rest.indexOf('/')
    const mod = useMods
      .getState()
      .mods.find(
        (candidate) =>
          candidate.key === rest.slice(0, cut) && candidate.uniqueId === rest.slice(cut + 1),
      )
    if (mod) {
      useMods.getState().openConfig(mod).catch(reportUnexpected)
    }
    return
  }
  if (id.startsWith('toggle-mod:')) {
    const rest = id.slice('toggle-mod:'.length)
    const cut = rest.indexOf('/')
    const mod = useMods
      .getState()
      .mods.find(
        (candidate) =>
          candidate.key === rest.slice(0, cut) && candidate.uniqueId === rest.slice(cut + 1),
      )
    if (mod) {
      useMods.getState().setEnabled(mod, !mod.enabled).catch(reportUnexpected)
    }
    return
  }
  if (id === 'action:play') {
    playOpenProfile()
    return
  }
  if (id === 'action:updates') {
    useUpdates.getState().load().catch(reportUnexpected)
    return
  }
  if (id === 'action:recent-changes') {
    useRecentChanges.getState().setOpen(true)
    return
  }
  if (id === 'action:downloads') {
    useQueue.getState().setOpen(true)
    return
  }
  if (id === 'action:downloads-folder') {
    openDownloadsDialog()
    return
  }
  if (id === 'action:import' || id === 'action:paste-link') {
    openImport({ profileId: useProfiles.getState().openId })
    return
  }
  if (id === 'action:collection-review') {
    const { openId } = useProfiles.getState()
    if (openId) {
      openImport({ profileId: openId, collectionUpdate: true })
    }
    return
  }
  if (id === 'action:find-crash-cause') {
    startCrashBisectFromPalette()
      .then((message) => {
        if (message) {
          useToasts.getState().push({ kind: 'info', title: message })
        }
      })
      .catch(reportUnexpected)
    return
  }
  if (id === 'action:share') {
    const { openId } = useProfiles.getState()
    if (openId) {
      openShare(openId)
    }
    return
  }
  if (id === 'action:new-profile') {
    useCommandPalette.getState().setCreating(true)
    return
  }
  if (id === 'action:stream-overlay') {
    const nav = useNav.getState()
    nav.closeSettings()
    nav.closeProfiles()
    nav.openGameSettings()
  }
}

export function runPaletteItem(id: string): void {
  useCommandPalette.getState().setOpen(false)
  if (id.startsWith('profile:')) {
    openProfile(id.slice('profile:'.length))
    return
  }
  if (id.startsWith('mod:')) {
    const rest = id.slice('mod:'.length)
    const cut = rest.indexOf('/')
    if (cut < 0) {
      return
    }
    const { openId } = useProfiles.getState()
    openModInProfile({
      profileId: openId,
      key: rest.slice(0, cut),
      uniqueId: rest.slice(cut + 1),
    })
    leaveShellPages()
    return
  }
  if (id.startsWith('settings:')) {
    const section = id.slice('settings:'.length)
    if (sections.has(section as SettingsSection)) {
      useNav.getState().openSettings(section as SettingsSection)
    }
    return
  }
  if (id.startsWith('shortcut:')) {
    const shortcut = id.slice('shortcut:'.length)
    if (SHORTCUTS.some((row) => row.id === shortcut)) {
      runShortcut(shortcut as ShortcutId)
    }
    return
  }
  runAction(id)
}
