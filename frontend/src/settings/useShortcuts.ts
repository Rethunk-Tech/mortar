import { useEffect } from 'react'
import { useCommandPalette } from '../commandPalette/store.ts'
import { useRenameRequest } from '../game/renameRequest.ts'
import { useTab } from '../game/tab.ts'
import { playOpenProfile } from '../launch/playOpen.ts'
import { requestFilterFocus } from '../mods/filterFocus.ts'
import { useUpdates } from '../mods/updates.ts'
import { openSettings, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { openImport, openShare } from '../share/store.ts'
import { dialogOpen, matchShortcut, shortcutAllowed } from './shortcuts.ts'

function runShortcut(id: NonNullable<ReturnType<typeof matchShortcut>>) {
  const profiles = useProfiles.getState()
  switch (id) {
    case 'command-palette':
      return useCommandPalette.getState().setOpen(true)
    case 'filter-mods':
      return requestFilterFocus()
    case 'play':
      return playOpenProfile()
    case 'check-updates':
      return useUpdates.getState().load()
    case 'open-settings':
      return openSettings()
    case 'tab-mods':
      return useTab.getState().setTab('mods')
    case 'tab-problems':
      return useTab.getState().setTab('problems')
    case 'tab-saves':
      return useTab.getState().setTab('saves')
    case 'tab-notes':
      return useTab.getState().setTab('notes')
    case 'tab-console':
      return useTab.getState().setTab('console')
    case 'tab-performance':
      return useTab.getState().setTab('performance')
    case 'new-profile':
      return profiles.create('New profile').catch(() => undefined)
    case 'duplicate-profile':
      return profiles.duplicate(profiles.openId).catch(() => undefined)
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
      return useTab.getState().setTab('console')
    default:
      return
  }
}

export function useAppShortcuts() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const id = matchShortcut(e)
      const typing = e.target instanceof HTMLElement ? e.target : null
      if (!(id && shortcutAllowed(id, typing, dialogOpen()))) {
        return
      }
      if (id === 'dismiss') {
        return
      }
      e.preventDefault()
      runShortcut(id)
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [])
}
