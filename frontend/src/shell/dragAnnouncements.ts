import type { Announcements, UniqueIdentifier } from '@dnd-kit/core'
import { useLingui } from '@lingui/react/macro'
import { useMemo } from 'react'

/**
 * What a screen reader hears while an item is dragged, by name: dnd-kit's own lines read out the raw ids (a folder
 * path, an entry key). Hovering the item's own place says nothing, so the pick-up line is not talked over at once.
 */
export function useDragAnnouncements(nameOf: (id: UniqueIdentifier) => string): Announcements {
  const { t } = useLingui()
  return useMemo(
    () => ({
      onDragStart: ({ active }) => t`Picked up ${{ name: nameOf(active.id) }}.`,
      onDragOver: ({ active, over }) => {
        if (over?.id === active.id) {
          return
        }
        const name = nameOf(active.id)
        return over
          ? t`${{ name }} is over ${{ place: nameOf(over.id) }}.`
          : t`${{ name }} is not over a place to drop it.`
      },
      onDragEnd: ({ active, over }) => {
        const name = nameOf(active.id)
        return over && over.id !== active.id
          ? t`Dropped ${{ name }} at ${{ place: nameOf(over.id) }}.`
          : t`Dropped ${{ name }} where it was.`
      },
      onDragCancel: ({ active }) => t`Moving ${{ name: nameOf(active.id) }} was cancelled.`,
    }),
    [t, nameOf],
  )
}
