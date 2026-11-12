import { Events } from '@wailsio/runtime'
import { useEffect, useEffectEvent } from 'react'

export type FolderEvent = 'library:mods-folder' | 'library:extra-folder' | 'library:downloads'

/** Runs onChange when the backend reports that a watched folder changed for this game. */
export function useFolderEvent(event: FolderEvent, game: string, onChange: () => void) {
  const changed = useEffectEvent(onChange)
  useEffect(
    () =>
      Events.On(event, (e) => {
        if (e.data === game) {
          changed()
        }
      }),
    [event, game],
  )
}
