import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'

export type PageHost =
  | 'nexus'
  | 'github'
  | 'thunderstore'
  | 'curseforge'
  | 'modrinth'
  | 'itch'
  | 'patreon'
  | 'vsmoddb'
  | 'paradox'
  | 'web'
  | ''
export type ModAction =
  | 'toggle'
  | 'details'
  | 'page'
  | 'files'
  | 'pin'
  | 'skip'
  | 'reason'
  | 'remove'

export interface MenuState {
  enabled: boolean
  host: PageHost
  pinned: boolean
  skipVersion: string
  hasUpdate: boolean
  // Mortar's own loader and bridge entries are always implicit, and a part another mod of its download requires is
  // implicit through that mod, so neither offers Mark items.
  bundled: boolean
  part: boolean
  implicit: boolean
}

// The site a mod's page is on, named when Mortar knows it.
export const hostOf = (url: string | undefined): PageHost => {
  if (!(url && URL.canParse(url))) {
    return ''
  }
  const host = new URL(url).hostname
  const on = (site: string) => host === site || host.endsWith(`.${site}`)
  if (on('github.com')) {
    return 'github'
  }
  if (on('nexusmods.com')) {
    return 'nexus'
  }
  if (on('thunderstore.io')) {
    return 'thunderstore'
  }
  if (on('curseforge.com')) {
    return 'curseforge'
  }
  if (on('modrinth.com')) {
    return 'modrinth'
  }
  if (on('patreon.com')) {
    return 'patreon'
  }
  if (on('mods.vintagestory.at')) {
    return 'vsmoddb'
  }
  if (on('mods.paradoxplaza.com')) {
    return 'paradox'
  }
  return on('itch.io') ? 'itch' : 'web'
}

// The action that opens a mod's page, named for its site.
export const openPageLabel = (i18n: I18n, host: PageHost): string =>
  i18n._(
    {
      github: msg`Open on GitHub`,
      nexus: msg`Open on Nexus`,
      thunderstore: msg`Open on Thunderstore`,
      curseforge: msg`Open on CurseForge`,
      modrinth: msg`Open on Modrinth`,
      itch: msg`Open on itch.io`,
      patreon: msg`Open on Patreon`,
      vsmoddb: msg`Open on the Mod DB`,
      paradox: msg`Open on Paradox Mods`,
      web: msg`Open page`,
      '': msg`Open page`,
    }[host],
  )

// The actions of a mod's menu, in menu order; Remove is the last and is set apart by a divider.
export function modActions(s: MenuState): ModAction[] {
  const out: ModAction[] = ['toggle', 'details']
  if (s.host !== '') {
    out.push('page')
  }
  out.push('files')
  out.push('pin')
  if (s.hasUpdate || s.skipVersion !== '') {
    out.push('skip')
  }
  if (!(s.bundled || s.part)) {
    out.push('reason')
  }
  out.push('remove')
  return out
}
