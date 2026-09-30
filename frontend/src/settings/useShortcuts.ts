import { useEffect } from 'react'
import { playOpenProfile } from '../launch/playOpen.ts'
import { requestFilterFocus } from '../mods/filterFocus.ts'
import { useUpdates } from '../mods/updates.ts'
import { openSettings } from '../nav/store.ts'
import { dialogOpen, matchShortcut, shortcutAllowed } from './shortcuts.ts'

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
      if (id === 'filter-mods') {
        requestFilterFocus()
        return
      }
      if (id === 'play') {
        playOpenProfile()
        return
      }
      if (id === 'check-updates') {
        useUpdates.getState().load()
        return
      }
      openSettings()
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [])
}
