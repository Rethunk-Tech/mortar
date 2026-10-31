import { useState } from 'react'

/** options with the selected ones first, each group in its own order. */
export function selectedFirst<T>(options: readonly T[], isSelected: (option: T) => boolean): T[] {
  return [...options.filter(isSelected), ...options.filter((o) => !isSelected(o))]
}

/**
 * A multi-select's options with what is picked listed first, so it is quick to unpick. The order is taken when the list
 * opens and kept while it is open: re-sorting on each pick would move the row under the pointer.
 */
export function useSelectedFirst<T>(options: readonly T[], isSelected: (option: T) => boolean) {
  const [opened, setOpened] = useState<T[] | null>(null)
  const current = opened !== null && opened.length === options.length ? opened : options
  return {
    options: current as T[],
    onOpen: () => setOpened(selectedFirst(options, isSelected)),
  }
}
