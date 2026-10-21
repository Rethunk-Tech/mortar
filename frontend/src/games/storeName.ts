import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'

// Store and launcher names are brands, shown as they are; only the words around them are translated.
const BRANDS: Record<string, string> = {
  steam: 'Steam',
  'flatpak-steam': 'Flatpak Steam',
  gog: 'GOG',
  lutris: 'Lutris',
  bottles: 'Bottles',
  ea: 'EA App',
  epic: 'Epic Games',
  xbox: 'Xbox',
  flatpak: 'Flatpak',
}
const VIA: Record<string, [store: string, launcher: string]> = {
  'gog-heroic': ['GOG', 'Heroic'],
  'gog-minigalaxy': ['GOG', 'Minigalaxy'],
  'epic-heroic': ['Epic', 'Heroic'],
}

// The store an install came from, as the window names it; "" for a folder the user chose.
export function storeName(store: string): string {
  const brand = BRANDS[store]
  if (brand) {
    return brand
  }
  const via = VIA[store]
  if (via) {
    const [name, launcher] = via
    return i18n._(msg`${name} via ${launcher}`)
  }
  return store === 'standalone' ? i18n._(msg`Default folder`) : ''
}
