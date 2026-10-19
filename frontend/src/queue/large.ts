import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'

/** The download the large-download modal asks about: the first one held, and how many more wait behind it. */
export interface LargeAsk {
  item: Item
  more: number
}

export function largeAsk(items: Item[]): LargeAsk | null {
  const held = items.filter((i) => i.state === 'confirm-large')
  const [item] = held
  return item === undefined ? null : { item, more: held.length - 1 }
}
