import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'

export type PageHost =
  | 'nexus'
  | 'github'
  | 'thunderstore'
  | 'curseforge'
  | 'modrinth'
  | 'itch'
  | 'web'
  | ''
export type ModAction = 'toggle' | 'details' | 'page' | 'files' | 'pin' | 'skip' | 'remove'

export interface MenuState {
  enabled: boolean
  host: PageHost
  pinned: boolean
  skipVersion: string
  hasUpdate: boolean
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
  out.push('remove')
  return out
}
