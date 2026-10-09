import type { MessageDescriptor } from '@lingui/core'
import { msg } from '@lingui/core/macro'

const NAMES: Record<string, MessageDescriptor> = {
  steam: msg`Steam`,
  'flatpak-steam': msg`Flatpak Steam`,
  gog: msg`GOG`,
  'gog-heroic': msg`GOG via Heroic`,
  'gog-minigalaxy': msg`GOG via Minigalaxy`,
  lutris: msg`Lutris`,
  bottles: msg`Bottles`,
  ea: msg`EA App`,
}

// The store an install came from, as the window names it; "" for a folder the user chose.
export const storeName = (store: string): MessageDescriptor | null => NAMES[store] ?? null
