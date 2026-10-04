import { Events } from '@wailsio/runtime'
import { useEffect, useRef } from 'react'

export type FolderEvent = 'library:mods-folder' | 'library:extra-folder' | 'library:downloads'

/** Runs onChange when the backend reports that a watched folder changed for this game. */
export function useFolderEvent(event: FolderEvent, game: string, onChange: () => void) {
  const latest = useRef(onChange)
  latest.current = onChange
  useEffect(() => {
    if (typeof Events.On !== 'function') {
      return
    }
    return Events.On(event, (e) => {
      if (e.data === game) {
        latest.current()
      }
    })
  }, [event, game])
}
