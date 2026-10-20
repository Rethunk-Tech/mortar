// The landing page's comparison: Mortar's side from the catalog and the code, each other manager's from its own
// repository. Counts come from rival-facts.json (refreshed each release); every other fact names the page it is read
// from, and the "Why Mortar?" pages cite the same sources. A fact a project does not document stays undefined and
// shows as "—".
import { rivalFacts } from './faq.ts'
import { modSites, playableGames } from './games.ts'

export interface Manager {
  id: string
  name: string
  logo: string
  games: string
  sites: string[]
  platforms: string[]
  shares: string[]
  opens: string[]
  imports: string[]
  account: string
  analytics?: string
  licence: string
  /** Where the facts about this manager come from. */
  sources: string[]
}

const count = (key: string) => String(rivalFacts()[key] ?? '')

export function managers(): Manager[] {
  return [
    {
      id: 'mortar',
      name: 'Mortar',
      logo: '/img/icon-192.png',
      games: String(playableGames().length),
      sites: modSites(),
      platforms: ['windows', 'linux', 'steamdeck'],
      shares: ['Link', 'File', 'LAN', 'Collection', 'Modpack'],
      opens: ['Codes', '.r2z', 'Collections'],
      imports: ['r2modman', 'gale', 'vortex', 'mod-organizer-2', 'stardrop'],
      account: 'None',
      analytics: 'None',
      licence: 'AGPL-3.0',
      sources: [],
    },
    {
      id: 'r2modman',
      name: 'r2modman',
      logo: '/img/managers/r2modman.png',
      games: count('r2modman_count'),
      sites: ['thunderstore'],
      platforms: ['windows', 'linux', 'steamdeck'],
      shares: ['Code', '.r2z'],
      opens: ['Own codes'],
      imports: [],
      account: 'None',
      licence: 'MIT',
      sources: [
        'https://github.com/ebkr/r2modmanPlus',
        'https://github.com/ebkr/r2modmanPlus/blob/develop/src/r2mm/mods/ProfileImportExport.ts',
      ],
    },
    {
      id: 'gale',
      name: 'Gale',
      logo: '/img/managers/gale.png',
      games: count('gale_count'),
      sites: ['thunderstore'],
      platforms: ['windows', 'linux'],
      shares: ['Code', 'File', 'Modpack'],
      opens: ['r2modman codes'],
      imports: ['r2modman', 'thunderstore-mod-manager'],
      account: 'None',
      licence: 'GPL-3.0',
      sources: [
        'https://github.com/Kesomannen/gale',
        'https://github.com/Kesomannen/gale/wiki/Features',
      ],
    },
    {
      id: 'vortex',
      name: 'Vortex',
      logo: '/img/managers/vortex.png',
      games: `${count('vortex_count')}+`,
      sites: ['nexus'],
      platforms: ['windows'],
      shares: ['Collection'],
      opens: ['Collections'],
      imports: [],
      account: 'Nexus, to download',
      analytics: 'Mixpanel, keyed to your Nexus id',
      licence: 'GPL-3.0',
      sources: [
        'https://github.com/Nexus-Mods/Vortex',
        'https://github.com/Nexus-Mods/Vortex/blob/master/docs/analytics.md',
      ],
    },
  ]
}

/** The managers a profile can be imported from, by id: their name and logo. */
export const IMPORT_FROM: Record<string, { name: string; logo?: string }> = {
  r2modman: { name: 'r2modman', logo: '/img/managers/r2modman.png' },
  gale: { name: 'Gale', logo: '/img/managers/gale.png' },
  vortex: { name: 'Vortex', logo: '/img/managers/vortex.png' },
  'mod-organizer-2': { name: 'Mod Organizer 2', logo: '/img/managers/mod-organizer-2.png' },
  stardrop: { name: 'Stardrop', logo: '/img/managers/stardrop.png' },
  'thunderstore-mod-manager': { name: 'Thunderstore Mod Manager' },
}
